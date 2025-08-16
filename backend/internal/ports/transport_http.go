package ports

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"hackload/internal/middleware"
	"hackload/internal/sqlc"
)

type HttpServer struct {
	queries *sqlc.Queries
	db      *sql.DB
}

func NewHttpServer(queries *sqlc.Queries, db *sql.DB) ServerInterface {
	return &HttpServer{
		queries: queries,
		db:      db,
	}
}

// Получить список бронирований
// (GET /api/bookings)
func (s *HttpServer) ListBookings(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.GetUserFromContext(r.Context())
	fmt.Printf("%#v\n", session)
	if !ok {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	bookings, err := s.queries.GetBookings(r.Context(), session.UserID)
	fmt.Printf("%#v, %v\n", bookings, err)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	for _, booking := range bookings {
		fmt.Printf("%#v\n", booking)
	}
}

// Создать бронирование
// (POST /api/bookings)
func (s *HttpServer) CreateBooking(w http.ResponseWriter, r *http.Request) {
	panic("not implemented") // TODO: Implement
}

// Отменить бронирование
// (PATCH /api/bookings/cancel)
func (s *HttpServer) CancelBooking(w http.ResponseWriter, r *http.Request) {
	panic("not implemented") // TODO: Implement
}

// Инициировать платеж для бронирования
// (PATCH /api/bookings/initiatePayment)
func (s *HttpServer) InitiatePayment(w http.ResponseWriter, r *http.Request) {
	panic("not implemented") // TODO: Implement
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
	panic("not implemented") // TODO: Implement
}

// Принимать уведомления от платежного шлюза
// (POST /api/payments/notifications)
func (s *HttpServer) OnPaymentUpdates(w http.ResponseWriter, r *http.Request) {
	panic("not implemented") // TODO: Implement
}

// Уведомить сервис, что платеж успешно проведен
// (GET /api/payments/success)
func (s *HttpServer) NotifyPaymentCompleted(w http.ResponseWriter, r *http.Request, params NotifyPaymentCompletedParams) {
	panic("not implemented") // TODO: Implement
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
