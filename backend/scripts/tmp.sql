
-- insert into bookings (user_id, event_id) values (1, 1);

insert into "seats" (
    event_id,
    external_id,
    row,
    number,
    price,
    status
) values (
    1,
    '4eba76de-4288-4b39-b65d-531efdc2fc62',
    1,
    1,
    '150.00',
    'FREE'
);

-- insert into booking_seats (user_id, booking_id, seat_id) values (1, 1, 1);
