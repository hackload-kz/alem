package ports

import "net/http"

type HttpServer struct{}

func NewHttpServer() ServerInterface {
	return &HttpServer{}
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
	panic("not implemented") // TODO: Implement
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
