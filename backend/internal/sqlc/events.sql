-- name: GetEventsList :many
select
    e.id,
    e.title
from events_archive e
where 1=1
    and (
        cast(sqlc.narg(query) as text) is null
        or e.id in (
            select f.id from events_archive_fts f
            where f.title match sqlc.narg(query)
            or f.description match sqlc.narg(query)
        )
        -- or e.title like '%' || cast(sqlc.narg(query) as text) || '%'
        -- or e.description like '%' || cast(sqlc.narg(query) as text) || '%'
    )
    and (
        cast(sqlc.narg(date) as date) is null 
        or cast(sqlc.narg(date) as date) = e.date_start
    )
limit sqlc.arg(limit)
offset sqlc.arg(offset)
;