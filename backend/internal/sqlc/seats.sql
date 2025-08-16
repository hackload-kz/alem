-- name: GetSeats :many
select
  *
from seats s
where 1=1
  and sqlc.arg(event_id) = s.event_id
  and (
    cast(sqlc.narg('row') as integer) is null
    or cast(sqlc.narg('row') as integer) = s.row 
  )
  and (
    cast(sqlc.narg('status') as text) is null
    or cast(sqlc.narg('status') as text) = s.status
  )
limit sqlc.arg(limit)
offset sqlc.arg(offset)
;

-- name: UpdateSeatStatus :exec
update seats 
set status = sqlc.arg(status)
where id = sqlc.arg(seat_id)
;

-- name: UpdateSeatsStatusByIDs :exec
update seats 
set status = sqlc.arg(status)
where id IN (sqlc.slice(seat_ids))
;

-- name: GetSeatByID :one
select * from seats
where id = sqlc.arg(seat_id)
;