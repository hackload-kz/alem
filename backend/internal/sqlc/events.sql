-- name: GetEventsList :many
select
    e.id,
    e.title
from events_archive e
where 1=1
    and (
        cast(sqlc.narg(query) as text) is null 
        or cast(sqlc.narg(query) as text) like '%' || e.title || '%'
        or cast(sqlc.narg(query) as text) like '%' || e.description || '%'
    )
    and (
        cast(sqlc.narg(date) as date) is null 
        or cast(sqlc.narg(date) as date) = e.date_start
    )
limit sqlc.arg(limit)
offset sqlc.arg(offset)
;
