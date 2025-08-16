package service

import "context"

type EventsService interface {
	GetList(ctx context.Context) (any, error)
}

type GetEventsList struct {
	// query (опциональный) - Параметр для полноконтекстного поиска
	Query *string

	// date (опциональный) - Параметр для фильтрации по дате события (формат: YYYY-MM-DD)
	Date *string

	// page (опциональный) - Номер страницы (минимум 1)
	Page *int

	// pageSize (опциональный) - Размер страницы (1-20)
	PageSize *int
}
