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

create table "bookings" (
    "id" integer primary key autoincrement,
    "user_id" integer not null references "users"("user_id"),
    "event_id" integer not null references "events_archive"("id"),

    -- статус: CREATED, PAYMENT_INITIATED, CONFIRMED, CANCELLED
    "status" text not null default 'CREATED'
);

create table "booking_seats" (
    "user_id" integer not null references "users"("user_id"),
    "booking_id" integer not null references "bookings"("id"),
    "seat_id" integer not null references "seats"("id"),

    -- композитный первичный ключ
    primary key ("booking_id", "seat_id"),
    
    -- уникальность места во всей системе
    unique ("seat_id")
);

create table "booking_payments" (
    "id" integer primary key autoincrement,
    "booking_id" integer not null references "bookings"("id"),
    "order_id" text not null,

    -- статус: INIT, SUCCESS, FAIL
    "status" text default 'INIT'
);

create table "booking_orders" (
    "id" integer primary key autoincrement,
    "booking_id" integer not null references "bookings"("id"),
    "order_id" text not null,

    -- статус: STARTED, SUBMITTED, CONFIRMED, CANCELLED
    "status" text default 'INIT'
);
