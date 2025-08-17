package sqlc

import (
	"context"

	sq "github.com/Masterminds/squirrel"
)

type GetEventsListParams struct {
	Query  *string
	Date   *string
	Offset int64
	Limit  int64
}

type GetEventsListRow struct {
	ID    int64
	Title *string
}

func (q *Queries) GetEventsList(ctx context.Context, arg GetEventsListParams) ([]GetEventsListRow, error) {
	query := sq.Select("e.id", "e.title").
		From("events_archive e").
		Where(sq.And{
			sq.Expr("1=1"),
		})

	if arg.Query != nil {
		query = query.Where(sq.And{
			sq.Expr("e.id in (select f.id from events_archive_fts f where f.title match ? or f.description match ?)", *arg.Query, *arg.Query),
		})
	}

	if arg.Date != nil {
		query = query.Where(sq.And{
			sq.Expr("? = e.date_start", *arg.Date),
		})
	}

	query = query.Limit(uint64(arg.Limit)).
		Offset(uint64(arg.Offset))

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, sql,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetEventsListRow
	for rows.Next() {
		var i GetEventsListRow
		if err := rows.Scan(&i.ID, &i.Title); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
