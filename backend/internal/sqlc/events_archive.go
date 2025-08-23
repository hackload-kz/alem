package sqlc

import (
	"context"
	"fmt"

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

func (q *Queries) GetEventsCount(ctx context.Context, arg GetEventsListParams) (int64, error) {
	query := sq.Select("COUNT(*)").
		From("events_archive f").
		Where(sq.Expr(`date_start = ?`, *arg.Date))

	sql, args, err := query.ToSql()
	if err != nil {
		return 0, err
	}
	row := q.db.QueryRowContext(ctx, sql, args...)

	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to scan count: %w", err)
	}

	return count, nil
}

func (q *Queries) GetEventsList(ctx context.Context, arg GetEventsListParams) ([]GetEventsListRow, error) {
	query := sq.Select("f.id", "f.title").
		From("events_archive_fts f")

	if arg.Query != nil {
		ftsQuery := fmt.Sprintf("title:%s OR description:%s", *arg.Query, *arg.Query)

		query = query.Where(sq.And{
			sq.Expr(`events_archive_fts MATCH ?`, ftsQuery),
		})
	}

	if arg.Date != nil {
		count, err := q.GetEventsCount(ctx, arg)
		if err != nil {
			return nil, fmt.Errorf("failed to get events count: %w", err)
		}

		if count == 0 {
			return nil, nil
		}

		query = query.Join("events_archive e on e.id = f.id").
			Where(sq.And{
				sq.Expr("e.date_start = ?", *arg.Date),
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
