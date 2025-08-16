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

-- name: CreateBooking :one
INSERT INTO bookings (user_id, event_id, status)
VALUES (sqlc.arg(user_id), sqlc.arg(event_id), 'CREATED')
RETURNING id;

-- name: CancelBooking :execresult
UPDATE bookings 
SET status = 'CANCELLED'
WHERE id = sqlc.arg(booking_id) 
  AND user_id = sqlc.arg(user_id)
  AND status IN ('CREATED', 'PAYMENT_INITIATED');
-- name: InsertBookingSeat :exec
insert into booking_seats (user_id, booking_id, seat_id)
values (sqlc.arg(user_id), sqlc.arg(booking_id), sqlc.arg(seat_id))
;

-- name: DeleteBookingSeat :execrows
delete from booking_seats 
where seat_id = sqlc.arg(seat_id)
  and user_id = sqlc.arg(user_id)
;

-- name: GetBooking :one
select * from bookings 
where id = sqlc.arg(booking_id)
;

-- name: GetBookingSeats :many
select seat_id from booking_seats
where booking_id = sqlc.arg(booking_id)
;

-- name: DeleteBookingSeats :execrows
delete from booking_seats 
where booking_id = sqlc.arg(booking_id)
;

-- name: UpdateBookingStatus :exec
UPDATE bookings 
SET status = sqlc.arg(status)
WHERE id = sqlc.arg(booking_id)
;

-- name: InsertBookingOrder :exec
INSERT INTO booking_orders (booking_id, order_id, status)
VALUES (sqlc.arg(booking_id), sqlc.arg(order_id), sqlc.arg(status))
;
