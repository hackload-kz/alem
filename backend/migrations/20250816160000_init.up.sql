create table "users" (
    "user_id" integer primary key,
    "email" text unique not null,
    "password_hash" text not null,
    "password_plain" text,  -- for testing purposes only, would not exist in production
    "first_name" text not null,
    "surname" text not null,
    "birthday" date,
    "registered_at" timestamp not null,
    "is_active" boolean not null,
    "last_logged_in" timestamp not null
);

CREATE INDEX idx_users_email ON users(email);

create table "events_archive" (
    "id" integer primary key,
    "title" text,
    "description" text,

    -- enum: 'film', 'cinema', 'stage', 'game'
    "type" text, 

    -- пример: 2025-12-15t20:00:00
    "datetime_start" timestamp not null, 

    -- используется для поиска
    -- пример: 2025-12-15
    "date_start" date generated always as (date("datetime_start")) stored,

    -- Enum: 'Билеттер', 'TicketRu', 'EventWorld', 'ShowTime'
    "provider" text
);

CREATE INDEX idx_events_date_type ON events_archive(date_start);
CREATE INDEX idx_events_title ON events_archive(title);
CREATE INDEX idx_events_description ON events_archive(description);

create table "seats" (
    "id" integer primary key autoincrement,
    "event_id" integer not null references "events_archive"("id"),

    -- id в ticket provider service
    "external_id" text,
    
    -- ряд: номер ряда (integer)
    "row" integer not null,
    
    -- номер: номер места в ряду (integer)
    "number" integer not null,

    -- пример: 15.00
    "price" text not null,
    
    -- статус: FREE, RESERVED, SOLD
    "status" text not null
);


CREATE INDEX idx_seats_event ON seats(event_id);
CREATE INDEX idx_seats_external ON seats(external_id) WHERE external_id IS NOT NULL;
CREATE INDEX idx_seats_event_row_status ON seats(event_id, status, row);
CREATE INDEX idx_seats_event_status_free ON seats(event_id, row, number) 
  WHERE status = 'FREE';

create table "bookings" (
    "id" integer primary key autoincrement,
    "user_id" integer not null references "users"("user_id"),
    "event_id" integer not null references "events_archive"("id"),

    -- статус: CREATED, PAYMENT_INITIATED, CONFIRMED, CANCELLED
    "status" text not null default 'CREATED'
);

CREATE INDEX idx_bookings_user ON bookings(user_id);
CREATE INDEX idx_bookings_event ON bookings(event_id);

create table "booking_seats" (
    "user_id" integer not null references "users"("user_id"),
    "booking_id" integer not null references "bookings"("id"),
    "seat_id" integer not null references "seats"("id"),

    -- композитный первичный ключ
    primary key ("booking_id", "seat_id"),
    
    -- уникальность места во всей системе
    unique ("seat_id")
);

CREATE INDEX idx_booking_seats_booking ON booking_seats(booking_id);
CREATE INDEX idx_booking_seats_user ON booking_seats(user_id);

create table "booking_payments" (
    "id" integer primary key autoincrement,
    "booking_id" integer not null references "bookings"("id"),
    "order_id" text not null,

    -- статус: INIT, SUCCESS, FAIL
    "status" text default 'INIT'
);

CREATE INDEX idx_booking_payments_order ON booking_payments(order_id);
CREATE INDEX idx_booking_payments_booking ON booking_payments(booking_id);

create table "booking_orders" (
    "id" integer primary key autoincrement,
    "booking_id" integer not null references "bookings"("id"),
    "order_id" text not null,

    -- статус: STARTED, SUBMITTED, CONFIRMED, CANCELLED
    "status" text default 'INIT'
);

CREATE INDEX idx_booking_orders_booking ON booking_orders(booking_id);
CREATE INDEX idx_booking_orders_order ON booking_orders(order_id);
