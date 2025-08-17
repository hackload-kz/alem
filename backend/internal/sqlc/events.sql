-- name: GetEventsList :many
select
    e.id,
    e.title
from events_archive e
where 1=1
    and (
        cast(sqlc.narg(query) as text) is null 
        or e.title like '%' || cast(sqlc.narg(query) as text) || '%'
        or e.description like '%' || cast(sqlc.narg(query) as text) || '%'
    )
    and (
        cast(sqlc.narg(date) as date) is null 
        or cast(sqlc.narg(date) as date) = e.date_start
    )
limit sqlc.arg(limit)
offset sqlc.arg(offset)
;

-- name: GetEventAnalytics :one
select
    COUNT(*) as total_seats,
    SUM(CASE WHEN s.status = 'SOLD' THEN 1 ELSE 0 END) as sold_seats,
    SUM(CASE WHEN s.status = 'RESERVED' THEN 1 ELSE 0 END) as reserved_seats,
    SUM(CASE WHEN s.status = 'FREE' THEN 1 ELSE 0 END) as free_seats,
    cast(
        (COALESCE(SUM(CASE WHEN s.status = 'SOLD' THEN CAST(s.price AS REAL) ELSE 0 END), 0))
        as real
    ) as total_revenue,
    (
        select COUNT(DISTINCT b.id) 
        from bookings b 
        where b.event_id = sqlc.arg(event_id)
    ) as bookings_count
from seats s
where s.event_id = sqlc.arg(event_id)
;
