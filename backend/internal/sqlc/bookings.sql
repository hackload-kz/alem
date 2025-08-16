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