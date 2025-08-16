
-- name: GetBookings :many
select
  b.id,
  b.event_id,
  json_group_array(
    json_object(
      'id', bs.seat_id
    )
  ) AS seats
from bookings b
join booking_seats as bs on bs.booking_id = b.id
where 1=1
  and sqlc.arg(user_id) = b.user_id
group by b.id, b.event_id;
