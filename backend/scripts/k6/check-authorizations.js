import { check } from 'k6';
import encoding from 'k6/encoding';
import http from 'k6/http';

const userCredentials = {
    1: {email: 'aysultan_talgat_1@fest.tix', password_plain: '/8eC$AD>'},
    2: {email: 'sultan_sultanov_2@show.go', password_plain: '*IVSf?kh)xa'},

    3: {email: 'aysultan_talgat_1@fest.tix', password_plain: '/8eC$AD>'},
    4: {email: 'ayaulym_bazarbaeva_3@quick.pass', password_plain: 'LDb60_%]4'},

    5: {email: 'sultan_sultanov_2@show.go', password_plain: '*IVSf?kh)xa'},
    6: {email: 'aysultan_talgat_1@fest.tix', password_plain: '/8eC$AD>'},

    7: {email: 'sultan_sultanov_2@show.go', password_plain: '*IVSf?kh)xa'},
    8: {email: 'ayaulym_bazarbaeva_3@quick.pass', password_plain: 'LDb60_%]4'},

    9: {email: 'ayaulym_bazarbaeva_3@quick.pass', password_plain: 'LDb60_%]4'},
    10: {email: 'aysultan_talgat_1@fest.tix', password_plain: '/8eC$AD>'},

    11: {email: 'ayaulym_bazarbaeva_3@quick.pass', password_plain: 'LDb60_%]4'},
    12: {email: 'sultan_sultanov_2@show.go', password_plain: '*IVSf?kh)xa'},
}

export const options = {
    vus: 1,
    iterations: 6,
    thresholds: {
        'http_reqs': ['count>=42'],
        'checks': ['rate>=1']
    },
};

export default function () {
    const BASE_URL = `${__ENV.API_URL}`;
    console.log(`Using: ${BASE_URL}`);
    console.log(`Iteration ${__ITER}`)
    const user1 = userCredentials[__ITER * 2 + 1];
    console.log(`user1: ${JSON.stringify(user1)}`)
    const username1 = user1.email;
    const password1 = user1.password_plain;
    const params1 = {
        headers: {
            'Authorization': `Basic ${encoding.b64encode(`${username1}:${password1}`)}`,
            'Content-Type': 'application/json'
        },
    }

    // hope that user 1 !== user 2
    const user2 = userCredentials[__ITER * 2 + 2];
    console.log(`user2: ${JSON.stringify(user2)}`)
    const username2 = user2.email;
    const password2 = user2.password_plain;
    const params2 = {
        headers: {
            'Authorization': `Basic ${encoding.b64encode(`${username2}:${password2}`)}`,
            'Content-Type': 'application/json'
        },
    }

    // user 1 lists places
    // user 1 decides to buy a ticket for the first available seat
    const seat = findNextFreeSeat(BASE_URL, params1);
    console.log(seat);
    // user 1 creates a new booking
    const createBookingRequest = {event_id: 1};
    const createBookingResponse = http.post(`${BASE_URL}/api/bookings`, JSON.stringify(createBookingRequest), params1);
    console.log(`Create booking: ${createBookingResponse.status}`);
    check(createBookingResponse, {
        'статус 200': (r) => r.status === 200 || r.status === 201
    });
    const booking = JSON.parse(createBookingResponse.body)
    console.log(booking);

    // user 1 selects the seat for the booking
    console.log(`Booking: ${booking.id}`)
    console.log(`Seat: ${seat.id}`)
    const selectSeatRequest = {booking_id: booking.id, seat_id: seat.id}
    const selectSeatResponse = http.patch(`${BASE_URL}/api/seats/select`, JSON.stringify(selectSeatRequest), params1)
    console.log(`Select seat ${selectSeatResponse.status}`);
    check(selectSeatResponse, {
        'статус 200': (r) => r.status === 200
    });

    // user 2 tries to release the selected seat buy user 1
    const releaseSeatRequest = {seat_id: seat.id};
    const releaseSeatResponse = http.patch(`${BASE_URL}/api/seats/release`, JSON.stringify(releaseSeatRequest), params2)
    console.log(`Release seat: ${releaseSeatResponse.status}`);
    check(releaseSeatResponse, {
        'статус 403': (r) => r.status === 403
    })

    // user 2 tries to cancel the user 1's booking
    const cancelBookingRequest = {booking_id: booking.id}
    const cancelBookingResponse = http.patch(`${BASE_URL}/api/bookings/cancel`, JSON.stringify(cancelBookingRequest), params2)
    console.log(`Cancel booking: ${cancelBookingResponse.status}`);
    check(cancelBookingResponse, {
        'статус 403': (r) => r.status === 403
    })

    // user 1 releases the selected seat buy user 1
    const releaseSeatRequestByOwner = {seat_id: seat.id};
    const releaseSeatResponseByOwner = http.patch(`${BASE_URL}/api/seats/release`, JSON.stringify(releaseSeatRequestByOwner), params1)
    console.log(`Release seat: ${releaseSeatResponseByOwner.status}`);
    check(releaseSeatResponseByOwner, {
        'статус 200': (r) => r.status === 200 || r.status === 201
    })

    // user 1 cancels the booking
    const cancelBookingRequestByOwner = {booking_id: booking.id}
    const cancelBookingResponseByOwner = http.patch(`${BASE_URL}/api/bookings/cancel`, JSON.stringify(cancelBookingRequestByOwner), params1)
    console.log(`Cancel booking: ${cancelBookingResponseByOwner.status}`);
    check(cancelBookingResponseByOwner, {
        'статус 200': (r) => r.status === 200 || r.status === 201
    })
}

function findNextFreeSeat(BASE_URL, params) {

    let seat
    let page = 1

    do {
        const listSeatsResponse = http.get(`${BASE_URL}/api/seats?event_id=1&page=${page++}`, params);
        console.log(`List places: ${listSeatsResponse.status}`);
        check(listSeatsResponse, {
            'статус 200': (r) => r.status === 200
        });
        console.log();
        const seats = JSON.parse(listSeatsResponse.body);

        console.log(seats)
        seat = seats.find(seat => seat.status === 'FREE');
    } while (!seat)

    return seat
}