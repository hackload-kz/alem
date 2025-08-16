package ports

import (
	"encoding/json"
	"net/http"

	"hackload/internal/sqlc"
)

type HttpServer struct {
	queries *sqlc.Queries
}

func NewHttpServer(queries *sqlc.Queries) ServerInterface {
	return &HttpServer{
		queries: queries,
	}
}

// Получить список бронирований
// (GET /api/bookings)
func (s *HttpServer) ListBookings(w http.ResponseWriter, r *http.Request) {
	panic("not implemented") // TODO: Implement
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
	panic("not implemented") // TODO: Implement
}

// Убрать место из брони
// (PATCH /api/seats/release)
func (s *HttpServer) ReleaseSeat(w http.ResponseWriter, r *http.Request) {
	panic("not implemented") // TODO: Implement
}

// Выбрать место для брони
// (PATCH /api/seats/select)
func (s *HttpServer) SelectSeat(w http.ResponseWriter, r *http.Request) {
	panic("not implemented") // TODO: Implement
}
