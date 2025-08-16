insert into users (user_id, email, password_hash, password_plain, first_name, surname, birthday, registered_at, is_active, last_logged_in) VALUES
(1, 'aysultan_talgat_1@fest.tix', 'bb8d44111d1dd3ba3107f0416c8b575ab1818955fd64034a53df75d256bf905e', '/8eC$AD>', 'Айсұлтан', 'Талғат', '2003-05-14', '2021-05-18 09:16:23', TRUE, '2022-04-24 09:16:23');

insert into events_archive (id, title, description, type, datetime_start, provider) VALUES
(1, 'Концерт мировой звезды Селеста Морейра из Лусарии', 'В нашу сказочную Көкдалу прилетает мировая музыкальная звезда Селеста Морейра из большой соседней страны Лусария. Ее выступление будет проходить на центральном стадионе, который вмещает в себя 100 тысяч зрителей.', 'concert', '2025-12-15T20:00:00', 'Билеттер'),
(2, 'Показ картины "Зеленая миля" с Владимир Смирнов в главной роли', 'Уникальное мероприятие для всей семьи. Не пропустите яркое шоу с участием известных артистов и незабываемые впечатления.', 'film', '2025-10-25T19:45:00', 'TicketRu');

insert into bookings (user_id, event_id) values (1, 1);

insert into "seats" (
    event_id,
    external_id,
    row,
    number,
    price,
    status
) values (
    1,
    'LOLKEK',
    1,
    1,
    '15.00',
    'FREE'
);

insert into booking_seats (user_id, booking_id, seat_id) values (1, 1, 1);