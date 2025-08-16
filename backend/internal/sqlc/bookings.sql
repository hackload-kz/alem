-- name: GetBookings :many
select
  b.id,
  b.event_id,
  CASE 
    WHEN COUNT(bs.seat_id) = 0 THEN cast(json_array() as text)
    ELSE cast(
      json_group_array(json_object('id', bs.seat_id))
      as text
    )
  END AS seats
from bookings b
left join booking_seats as bs on bs.booking_id = b.id
where 1=1
  and sqlc.arg(user_id) = b.user_id
group by b.id, b.event_id;

-- name: InsertBookingSeat :exec
insert into booking_seats (user_id, booking_id, seat_id)
values (sqlc.arg(user_id), sqlc.arg(booking_id), sqlc.arg(seat_id))
;

-- name: DeleteBookingSeat :execrows
delete from booking_seats 
where seat_id = sqlc.arg(seat_id)
  and user_id = sqlc.arg(user_id)
;