-- name: GetSeats :many
select
  *
from seats s
where 1=1
  and sqlc.arg(event_id) = s.event_id
  and (
    sqlc.narg('row') = s.row 
    or sqlc.narg('row') is null
  )
  and (
    sqlc.narg('status') = s.status
    or sqlc.narg('status') is null
  )
limit sqlc.arg(limit)
offset sqlc.arg(offset)
;