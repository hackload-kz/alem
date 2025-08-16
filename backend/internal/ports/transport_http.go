package ports

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"hackload/internal/middleware"
	"hackload/internal/portriver"
	"hackload/internal/sqlc"

	"github.com/riverqueue/river"
)

type HttpServer struct {
	queries     *sqlc.Queries
	db          *sql.DB
	riverClient *river.Client[*sql.Tx]
}

func NewHttpServer(queries *sqlc.Queries, db *sql.DB, riverClient *river.Client[*sql.Tx]) ServerInterface {
	return &HttpServer{
		queries:     queries,
		db:          db,
		riverClient: riverClient,
	}
}

// Получить список бронирований
// (GET /api/bookings)
// Возвращает массив Bookings с массивом seats
// ```
// [
//
//	{
//	  "id": 456,
//	  "event_id": 123,
//	  "seats": [
//	    {"id": 789}
//	  ]
//	}
//
// ]
// ```
func (s *HttpServer) ListBookings(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		fmt.Println("ERROR: middleware.GetUserFromContext: false")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	bookings, err := s.queries.GetBookings(r.Context(), session.UserID)
	if err != nil {
		fmt.Println("ERROR: s.queries.GetBookings:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	response := make(ListBookingsResponse, 0, len(bookings))
	for _, booking := range bookings {
		var seats []ListEventsResponseItemSeat
		if err := json.Unmarshal([]byte(booking.Seats), &seats); err != nil {
			fmt.Println("ERROR: json.Unmarshal seats:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		item := ListBookingsResponseItem{
			Id:      booking.ID,
			EventId: booking.EventID,
			Seats:   &seats,
		}
		response = append(response, item)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// Создать бронирование
// (POST /api/bookings)
// При создании необходимо возвращать 201
func (s *HttpServer) CreateBooking(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		fmt.Println("ERROR: middleware.GetUserFromContext: false")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var req CreateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Println("ERROR: json.NewDecoder:", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	bookingID, err := s.queries.CreateBooking(r.Context(), sqlc.CreateBookingParams{
		UserID:  session.UserID,
		EventID: req.EventId,
	})
	if err != nil {
		fmt.Println("ERROR: s.queries.CreateBooking:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	statusEq := "CREATED"

	if _, err = s.riverClient.Insert(
		r.Context(),
		portriver.ReleaseSeatsArgs{
			BookingID: bookingID,
			StatusEq:  &statusEq,
		},
		&river.InsertOpts{ScheduledAt: time.Now().UTC().Add(10 * time.Minute)},
	); err != nil {
		fmt.Println("ERROR: s.riverClient.Insert:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	response := CreateBookingResponse{
		Id: bookingID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// Отменить бронирование
// (PATCH /api/bookings/cancel)
func (s *HttpServer) CancelBooking(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		fmt.Println("ERROR: middleware.GetUserFromContext: false")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var req CancelBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Println("ERROR: json.NewDecoder:", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "Could not start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)

	// 1. GetBooking to verify it exists
	booking, err := qtx.GetBooking(r.Context(), req.BookingId)
	if err != nil {
		return
	}

	if booking.Status == "PAYMENT_INITIATED" || booking.Status == "CANCELLED" {
		return
	}

	if _, err = qtx.CancelBooking(r.Context(), sqlc.CancelBookingParams{
		BookingID: req.BookingId,
		UserID:    session.UserID,
	}); err != nil {
		fmt.Println("ERROR: s.queries.CancelBooking:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	/*
		1. Если CONFIRMED -> Отменить в TicketProvider -> Освободить места
		2. ЕСЛИ CONFIRMED -> Вернуть деньги в Payment Gateway
	*/
	if booking.Status == "CONFIRMED" {
		// TODO
	}

	// Если CREATED -> Освободить места
	if booking.Status == "CONFIRMED" || booking.Status == "CREATED" {
		if _, err := s.riverClient.Insert(
			r.Context(),
			&portriver.ReleaseSeatsArgs{
				BookingID: req.BookingId,
				StatusEq:  &booking.Status,
			},
			nil,
		); err != nil {
			fmt.Println("ERROR: s.riverClient.Insert:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, "Could not commit transaction", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Инициировать платеж для бронирования
// (PATCH /api/bookings/initiatePayment)
func (s *HttpServer) InitiatePayment(w http.ResponseWriter, r *http.Request) {
	/*
		1. Создать платеж в PaymentGateway
		2. Создать booking_payments с указанным order_id
		3. Изменить статуст на PAYMENT_INITIATED
		4. Вернуть URL оплаты
	*/
}

// Получить список событий
// (GET /api/events)
func (s *HttpServer) ListEvents(w http.ResponseWriter, r *http.Request, params ListEventsParams) {
	page := int64(1)
	pageSize := int64(10)

	if params.Page != nil && *params.Page > 0 {
		page = int64(*params.Page)
	}
	if params.PageSize != nil && *params.PageSize > 0 && *params.PageSize <= 20 {
		pageSize = int64(*params.PageSize)
	}

	offset := (page - 1) * pageSize

	var dateStr *string
	if params.Date != nil {
		dateString := params.Date.String()
		dateStr = &dateString
	}

	events, err := s.queries.GetEventsList(r.Context(), sqlc.GetEventsListParams{
		Query:  params.Query,
		Date:   dateStr,
		Offset: offset,
		Limit:  pageSize,
	})
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	response := make(ListEventsResponse, 0, len(events))
	for _, event := range events {
		eventItem := ListEventsResponseItem{
			Id: event.ID,
		}
		if event.Title != nil {
			eventItem.Title = *event.Title
		}
		response = append(response, eventItem)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// Уведомить сервис, что платеж неуспешно проведен
// (GET /api/payments/fail)
func (s *HttpServer) NotifyPaymentFailed(w http.ResponseWriter, r *http.Request, params NotifyPaymentFailedParams) {
	/*

		1. Изменить статус booking_payments на FAIL
		2. Изменить статуc bookings на CANCELLED
		3. Отменить в TicketProvider -> Освободить места

	*/
}

// Принимать уведомления от платежного шлюза
// (POST /api/payments/notifications)
func (s *HttpServer) OnPaymentUpdates(w http.ResponseWriter, r *http.Request) {
}

// Уведомить сервис, что платеж успешно проведен
// (GET /api/payments/success)
func (s *HttpServer) NotifyPaymentCompleted(w http.ResponseWriter, r *http.Request, params NotifyPaymentCompletedParams) {
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "Could not start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)

	// 1. Update booking_payments status to SUCCESS
	err = qtx.UpdateBookingPaymentStatus(r.Context(), sqlc.UpdateBookingPaymentStatusParams{
		Status:  stringPtr("SUCCESS"),
		OrderID: fmt.Sprintf("%d", params.OrderId),
	})
	if err != nil {
		fmt.Printf("ERROR: failed to update payment status: %v\n", err)
		http.Error(w, "Failed to update payment status", http.StatusInternalServerError)
		return
	}

	// 2. Get booking by payment order ID
	booking, err := qtx.GetBookingByPaymentOrderID(r.Context(), fmt.Sprintf("%d", params.OrderId))
	if err != nil {
		fmt.Printf("ERROR: failed to get booking by payment order ID: %v\n", err)
		http.Error(w, "Booking not found", http.StatusNotFound)
		return
	}

	// 3. Update booking status to CONFIRMED
	err = qtx.UpdateBookingStatus(r.Context(), sqlc.UpdateBookingStatusParams{
		Status:    "CONFIRMED",
		BookingID: booking.ID,
	})
	if err != nil {
		fmt.Printf("ERROR: failed to update booking status: %v\n", err)
		http.Error(w, "Failed to update booking status", http.StatusInternalServerError)
		return
	}

	// Commit database changes first
	if err = tx.Commit(); err != nil {
		http.Error(w, "Could not commit transaction", http.StatusInternalServerError)
		return
	}

	// 4. Trigger ConfirmBookingWorker to handle EventProvider confirmation and seat updates
	if _, err = s.riverClient.Insert(r.Context(), portriver.ConfirmBookingArgs{
		BookingID: booking.ID,
	}, nil); err != nil {
		fmt.Printf("ERROR: failed to queue ConfirmBookingWorker: %v\n", err)
		// Don't fail the request - payment was already processed successfully
	}

	w.WriteHeader(http.StatusOK)
}

func stringPtr(s string) *string {
	return &s
}

// Получить список мест
// (GET /api/seats)
func (s *HttpServer) ListSeats(w http.ResponseWriter, r *http.Request, params ListSeatsParams) {
	page := int64(1)
	pageSize := int64(10)

	if params.Page != nil && *params.Page > 0 {
		page = int64(*params.Page)
	}
	if params.PageSize != nil && *params.PageSize > 0 && *params.PageSize <= 20 {
		pageSize = int64(*params.PageSize)
	}

	offset := (page - 1) * pageSize

	var rowFilter *int64
	if params.Row != nil && *params.Row > 0 {
		tmp := int64(*params.Row)
		rowFilter = &tmp
	}

	var statusFilter *string
	if params.Status != nil {
		statusStr := string(*params.Status)
		statusFilter = &statusStr
	}

	seats, err := s.queries.GetSeats(r.Context(), sqlc.GetSeatsParams{
		EventID: params.EventId,
		Row:     rowFilter,
		Status:  statusFilter,
		Offset:  offset,
		Limit:   pageSize,
	})
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	response := make(ListSeatsResponse, 0, len(seats))
	for _, seat := range seats {
		seatItem := ListSeatsResponseItem{
			Id:     seat.ID,
			Number: seat.Number,
			Price:  seat.Price,
			Row:    seat.Row,
			Status: ListSeatsResponseItemStatus(seat.Status),
		}
		response = append(response, seatItem)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// Убрать место из брони
// (PATCH /api/seats/release)
func (s *HttpServer) ReleaseSeat(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req ReleaseSeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "Could not start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)

	rowsAffected, err := qtx.DeleteBookingSeat(r.Context(), sqlc.DeleteBookingSeatParams{
		SeatID: req.SeatId,
		UserID: session.UserID,
	})
	if err != nil {
		http.Error(w, "Could not release seat", 419)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Seat not found in your bookings", 419)
		return
	}

	err = qtx.UpdateSeatStatus(r.Context(), sqlc.UpdateSeatStatusParams{
		Status: "FREE",
		SeatID: req.SeatId,
	})
	if err != nil {
		http.Error(w, "Could not update seat status", 419)
		return
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, "Could not commit transaction", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Выбрать место для брони
// (PATCH /api/seats/select)
func (s *HttpServer) SelectSeat(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req SelectSeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "Could not start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)

	err = qtx.InsertBookingSeat(r.Context(), sqlc.InsertBookingSeatParams{
		UserID:    session.UserID,
		BookingID: req.BookingId,
		SeatID:    req.SeatId,
	})
	if err != nil {
		http.Error(w, "Could not select seat", 419)
		return
	}

	err = qtx.UpdateSeatStatus(r.Context(), sqlc.UpdateSeatStatusParams{
		Status: "RESERVED",
		SeatID: req.SeatId,
	})
	if err != nil {
		http.Error(w, "Could not update seat status", 419)
		return
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, "Could not commit transaction", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
