import { check, sleep } from 'k6';
import encoding from 'k6/encoding';
import http from 'k6/http';
import { Counter, Rate } from 'k6/metrics';

// Custom metrics for tracking bookings
const successfulBookings = new Counter('successful_bookings');
const failedBookings = new Counter('failed_bookings');
const conflictBookings = new Counter('conflict_bookings');
const failedSeatRequests = new Counter('failed_seat_requests');

// Rate metric for Prometheus - tracks booking success rate
const bookingSuccessRate = new Rate('booking_success_rate');

export const options = {
  scenarios: {
    // Reset scenario - runs first and only once
    reset_system: {
      executor: 'shared-iterations',
      vus: 1,
      iterations: 1,
      maxDuration: '30s',
      exec: 'resetSystem',
      startTime: '0s',
    },
    // Main booking test - starts after reset
    biletter_booking_test: {
      executor: 'per-vu-iterations',
      vus: 10,
      iterations: 1000, // 3 users x 1000 iterations = 3000 booking attempts
      maxDuration: '30m',
      startTime: '30s', // Start 30 seconds after reset
    },
  },
};

const users = [
  { email: 'aysultan_talgat_1@fest.tix', password: '/8eC$AD>' },
  { email: 'ayaulym_bazarbaeva_3@quick.pass', password: 'LDb60_%]4' },
  { email: 'muslima_serikbaeva_999996@ticket.world', password: 'U|t,P#R~' },
  { email: 'ayan_asqar_999997@fest.tix', password: '_lnAio:Br7b' },
  { email: 'kozayym_asanova_999999@hackhload.kz', password: '_^@5=[oc!)+$' },
  { email: 'alinur_aytzhanov_1000000@ticket.world', password: 'bjD[wU=u#@Wk1\ED' },
  { email: 'sofiya_karimova_999989@show.go', password: 't~bv:18I\V$@' },
  { email: 'aysha_sarsenova_999990@ticket.world', password: 'zUKhVyYs%>\Ww' },
  { email: 'medina_rakhmetova_999991@show.go', password: ')~c^V+Tx#G9' },
  { email: 'іnzhu_ospanova_372921@quick.pass', password: 'lLZ~Z9X+>|C&(' }
];

function createBasicAuthHeader(user) {
  const credentials = `${user.email}:${user.password}`;
  const encodedCredentials = encoding.b64encode(credentials);
  return `Basic ${encodedCredentials}`;
}

// Reset function - runs once before all other scenarios
export function resetSystem() {
  const baseUrl = __ENV.API_URL;
  
  console.log('🔄 Resetting system before booking tests...');
  
  const resetResponse = http.post(`${baseUrl}/api/reset`, null, {
    headers: {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
      'User-Agent': 'K6-Booking-Test-Reset/1.0'
    },
    timeout: '30s'
  });

  check(resetResponse, {
    'reset system status is 200 or 204': (r) => r.status === 200 || r.status === 204,
    'reset completed successfully': (r) => r.status < 300,
  });

  if (resetResponse.status >= 200 && resetResponse.status < 300) {
    console.log('✅ System reset completed successfully');
  } else {
    console.error(`❌ System reset failed: ${resetResponse.status} - ${resetResponse.body}`);
    // Don't fail the test, just log the error
  }

  // Small delay to ensure reset is fully processed
  sleep(2);
}

export default function () {
  const baseUrl = __ENV.API_URL;
  // Use modulo to cycle through available users safely
  const userIndex = (__VU - 1) % users.length;
  const user = users[userIndex];
  
  const params = {
    headers: {
      'Authorization': createBasicAuthHeader(user),
      'Accept': 'application/json',
      'Content-Type': 'application/json'
    },
    timeout: '30s'
  };

  // Step 1: Use event with ID 1 (hardcoded as per requirements)
  const eventId = 1;

  // Step 3: Gather information about free places
  const seatsResponse = http.get(
    `${baseUrl}/api/seats?event_id=${eventId}&status=FREE&pageSize=5`,
    params
  );

  check(seatsResponse, {
    'get free seats status is 200': (r) => r.status === 200,
    'seats response has valid structure': (r) => {
      try {
        const data = JSON.parse(r.body);
        return Array.isArray(data);
      } catch {
        return false;
      }
    },
    'return number of seats <= pageSize': (r) => {
      try {
        const data = JSON.parse(r.body);
        return Array.isArray(data) && data.length <= 5; // pageSize=5 from URL
      } catch {
        return false;
      }
    },
    'all seats have status FREE': (r) => {
      try {
        const data = JSON.parse(r.body);
        if (!Array.isArray(data)) return false;
        return data.every(seat => seat.status === 'FREE');
      } catch {
        return false;
      }
    },
    'all seat IDs are unique': (r) => {
      try {
        const data = JSON.parse(r.body);
        if (!Array.isArray(data)) return false;
        const ids = data.map(seat => seat.id);
        return ids.length === new Set(ids).size;
      } catch {
        return false;
      }
    },
    'seats have required format': (r) => {
      try {
        const data = JSON.parse(r.body);
        if (!Array.isArray(data) || data.length === 0) return true; // Empty array is valid
        
        return data.every(seat => 
          typeof seat.id === 'number' &&
          typeof seat.row === 'number' &&
          typeof seat.number === 'number' &&
          typeof seat.status === 'string' &&
          typeof seat.price === 'string'
        );
      } catch {
        return false;
      }
    },
  });

  if (seatsResponse.status !== 200) {
    failedSeatRequests.add(1);
    console.error(`❌ Get seats failed: ${seatsResponse.status} - ${seatsResponse.body}`);
    return;
  }

  let freeSeats;
  try {
    freeSeats = JSON.parse(seatsResponse.body);
  } catch {
    console.error(`❌ Failed to parse seats response`);
    return;
  }

  if (!Array.isArray(freeSeats) || freeSeats.length === 0) {
    console.error(`❌ No free seats available`);
    return;
  }

  // Step 4: Create a booking
  const createBookingPayload = {
    event_id: eventId
  };

  const bookingResponse = http.post(
    `${baseUrl}/api/bookings`,
    JSON.stringify(createBookingPayload),
    params
  );

  check(bookingResponse, {
    'create booking status is 201': (r) => r.status === 201,
    'booking response has id': (r) => {
      try {
        const data = JSON.parse(r.body);
        return typeof data.id === 'number';
      } catch {
        return false;
      }
    },
  });

  if (bookingResponse.status !== 201) {
    console.error(`❌ Create booking failed: ${bookingResponse.status} - ${bookingResponse.body}`);
    return;
  }

  let booking;
  try {
    booking = JSON.parse(bookingResponse.body);
  } catch {
    console.error(`❌ Failed to parse booking response`);
    return;
  }

  // Step 5: Take the first free place
  const firstFreeSeat = freeSeats[0];
  const selectSeatPayload = {
    booking_id: booking.id,
    seat_id: firstFreeSeat.id
  };

  const selectSeatResponse = http.patch(
    `${baseUrl}/api/seats/select`,
    JSON.stringify(selectSeatPayload),
    params
  );

  check(selectSeatResponse, {
    'select seat status is 200': (r) => r.status === 200,
  });

  if (selectSeatResponse.status === 200) {
    successfulBookings.add(1);
    bookingSuccessRate.add(true);
    console.log(`✅ User ${user.email} successfully booked seat ${firstFreeSeat.id} (row ${firstFreeSeat.row}, number ${firstFreeSeat.number}) [Total: ${successfulBookings.count}]`);
    
    // Verify booking confirmation using GET /api/bookings
    sleep(0.5); // Brief delay to ensure booking is processed
    const listBookingsResponse = http.get(`${baseUrl}/api/bookings`, params);
    
    check(listBookingsResponse, {
      'list bookings status is 200': (r) => r.status === 200,
      'booking contains selected seat': (r) => {
        try {
          const bookings = JSON.parse(r.body);
          if (!Array.isArray(bookings)) return false;
          
          // Find our booking and verify it has the selected seat
          const ourBooking = bookings.find(b => b.id === booking.id);
          if (!ourBooking) return false;
          
          // Check if the booking has seats and contains our target seat
          if (!Array.isArray(ourBooking.seats)) return false;
          return ourBooking.seats.some(seat => seat.id === firstFreeSeat.id);
        } catch {
          return false;
        }
      },
      'booking has correct event_id': (r) => {
        try {
          const bookings = JSON.parse(r.body);
          if (!Array.isArray(bookings)) return false;
          
          const ourBooking = bookings.find(b => b.id === booking.id);
          return ourBooking && ourBooking.event_id === eventId;
        } catch {
          return false;
        }
      },
    });
    
    try {
      const bookings = JSON.parse(listBookingsResponse.body);
      const ourBooking = bookings.find(b => b.id === booking.id);
      if (ourBooking && ourBooking.seats && ourBooking.seats.length > 0) {
        console.log(`🎫 User ${user.email} booking confirmed with ${ourBooking.seats.length} seat(s) in GET /api/bookings`);
      }
    } catch {
      console.error(`❌ User ${user.email} failed to verify booking in GET /api/bookings`);
    }
    
  } else if (selectSeatResponse.status === 419) {
    conflictBookings.add(1);
    bookingSuccessRate.add(false);
    console.log(`⚠️ User ${user.email} failed to book seat ${firstFreeSeat.id} - seat already taken [Conflicts: ${conflictBookings.count}]`);
  } else {
    failedBookings.add(1);
    bookingSuccessRate.add(false);
    console.error(`❌ User ${user.email} select seat failed: ${selectSeatResponse.status} - ${selectSeatResponse.body} [Failures: ${failedBookings.count}]`);
  }

  sleep(Math.random() * 2 + 0.5);
}

// Conflict test function - two users try to book the same seat
export function conflictTest() {
  const baseUrl = __ENV.API_URL;
  // Use modulo to cycle through available users safely
  const userIndex = (__VU - 1) % users.length;
  const user = users[userIndex];
  
  const params = {
    headers: {
      'Authorization': createBasicAuthHeader(user),
      'Accept': 'application/json',
      'Content-Type': 'application/json'
    },
    timeout: '30s'
  };

  const eventId = 1;

  // Both users get free seats
  const seatsResponse = http.get(
    `${baseUrl}/api/seats?event_id=${eventId}&page=1&pageSize=20&row=5&status=FREE`,
    params
  );

  check(seatsResponse, {
    'conflict test get free seats status is 200': (r) => r.status === 200,
  });

  if (seatsResponse.status !== 200) {
    failedSeatRequests.add(1);
    console.error(`❌ Conflict test get seats failed: ${seatsResponse.status}`);
    return;
  }

  let freeSeats;
  try {
    freeSeats = JSON.parse(seatsResponse.body);
  } catch {
    console.error(`❌ Conflict test failed to parse seats response`);
    return;
  }

  if (!Array.isArray(freeSeats) || freeSeats.length === 0) {
    console.error(`❌ Conflict test no free seats available`);
    return;
  }

  // Both users create bookings
  const createBookingPayload = {
    event_id: eventId
  };

  const bookingResponse = http.post(
    `${baseUrl}/api/bookings`,
    JSON.stringify(createBookingPayload),
    params
  );

  check(bookingResponse, {
    'conflict test create booking status is 201': (r) => r.status === 201,
  });

  if (bookingResponse.status !== 201) {
    console.error(`❌ Conflict test create booking failed: ${bookingResponse.status}`);
    return;
  }

  let booking;
  try {
    booking = JSON.parse(bookingResponse.body);
  } catch {
    console.error(`❌ Conflict test failed to parse booking response`);
    return;
  }

  // Both users try to book THE SAME SEAT (first free seat)
  const targetSeat = freeSeats[0];
  const selectSeatPayload = {
    booking_id: booking.id,
    seat_id: targetSeat.id
  };

  console.log(`🎯 User ${user.email} attempting to book seat ${targetSeat.id} (row ${targetSeat.row}, number ${targetSeat.number})`);

  // Small random delay to simulate real-world timing variations
  sleep(Math.random() * 0.1);

  const selectSeatResponse = http.patch(
    `${baseUrl}/api/seats/select`,
    JSON.stringify(selectSeatPayload),
    params
  );

  check(selectSeatResponse, {
    'conflict test seat response is valid': (r) => r.status === 200 || r.status === 419,
  });

  if (selectSeatResponse.status === 200) {
    successfulBookings.add(1);
    bookingSuccessRate.add(true);
    console.log(`✅ User ${user.email} successfully booked seat ${targetSeat.id} - WINNER! [Total: ${successfulBookings.count}]`);
    check(selectSeatResponse, {
      'conflict test winner gets seat': (r) => r.status === 200,
    });

    // Verify booking confirmation using ListBookings
    sleep(0.5); // Brief delay to ensure booking is processed
    const listBookingsResponse = http.get(`${baseUrl}/api/bookings`, params);
    
    check(listBookingsResponse, {
      'conflict test list bookings status is 200': (r) => r.status === 200,
      'conflict test booking is confirmed': (r) => {
        try {
          const bookings = JSON.parse(r.body);
          if (!Array.isArray(bookings)) return false;
          
          // Find our booking and verify it has the selected seat
          const ourBooking = bookings.find(b => b.id === booking.id);
          if (!ourBooking) return false;
          
          // Check if the booking has seats and contains our target seat
          if (!Array.isArray(ourBooking.seats)) return false;
          return ourBooking.seats.some(seat => seat.id === targetSeat.id);
        } catch {
          return false;
        }
      },
    });

    try {
      const bookings = JSON.parse(listBookingsResponse.body);
      const ourBooking = bookings.find(b => b.id === booking.id);
      if (ourBooking && ourBooking.seats && ourBooking.seats.length > 0) {
        console.log(`🎫 User ${user.email} booking confirmed with ${ourBooking.seats.length} seat(s)`);
      }
    } catch {
      console.error(`❌ User ${user.email} failed to verify booking confirmation`);
    }

  } else if (selectSeatResponse.status === 419) {
    conflictBookings.add(1);
    bookingSuccessRate.add(false);
    console.log(`⚠️ User ${user.email} failed to book seat ${targetSeat.id} - seat already taken (EXPECTED BEHAVIOR) [Conflicts: ${conflictBookings.count}]`);
    check(selectSeatResponse, {
      'conflict test loser gets 419 status': (r) => r.status === 419,
    });

    // Verify that loser's booking has no seats
    sleep(0.5);
    const listBookingsResponse = http.get(`${baseUrl}/api/bookings`, params);
    
    check(listBookingsResponse, {
      'conflict test loser list bookings status is 200': (r) => r.status === 200,
      'conflict test loser booking has no seats': (r) => {
        try {
          const bookings = JSON.parse(r.body);
          if (!Array.isArray(bookings)) return false;
          
          const ourBooking = bookings.find(b => b.id === booking.id);
          if (!ourBooking) return true; // Booking might not exist, which is also valid
          
          // If booking exists, it should have no seats or empty seats array
          return !ourBooking.seats || ourBooking.seats.length === 0;
        } catch {
          return false;
        }
      },
    });

    console.log(`📋 User ${user.email} booking correctly has no confirmed seats`);

  } else {
    failedBookings.add(1);
    bookingSuccessRate.add(false);
    console.error(`❌ User ${user.email} conflict test unexpected response: ${selectSeatResponse.status} - ${selectSeatResponse.body} [Failures: ${failedBookings.count}]`);
  }

  sleep(0.5);
}

export function setup() {
  console.log(`🚀 Starting Biletter booking test with ${users.length} users`);
  
  // Safely access options to avoid undefined errors
  try {
    if (options && options.scenarios) {
      const mainScenario = options.scenarios.biletter_booking_test;
      const conflictScenario = options.scenarios.conflict_test;
      
      if (mainScenario && mainScenario.vus && mainScenario.iterations) {
        console.log(`📊 Main scenario: ${mainScenario.vus * mainScenario.iterations} total booking attempts`);
      }
      
      
      if (options.scenarios.reset_system) {
        console.log(`🔄 Reset scenario: System will be reset before tests begin`);
      }
    }
  } catch (error) {
    console.log(`⚠️ Could not access scenario details: ${error.message}`);
  }
  
  console.log(`🎯 Test environment: biletter_booking_test`);
  
  return {
    startTime: Date.now(),
    testVersion: 'v1.0.0',
    environment: 'biletter_booking_test'
  };
}

export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  console.log(`✅ Biletter Booking Test completed in ${duration}s`);
  console.log(`📊 BOOKING STATISTICS:`);
  console.log(`   ✅ Successful bookings: ${successfulBookings.count || 0}`);
  console.log(`   ❌ Failed bookings: ${failedBookings.count || 0}`);
  console.log(`   ⚠️ Conflict bookings: ${conflictBookings.count || 0}`);
  console.log(`   🔍 Failed seat requests: ${failedSeatRequests.count || 0}`);
  console.log(`   📊 Total attempts: ${(successfulBookings.count || 0) + (conflictBookings.count || 0) + (failedBookings.count || 0)}`);
  console.log(`📈 Check metrics for booking performance analysis`);
  console.log(`   Target: ${data.environment}`);
  console.log(`   Scenarios tested: system reset, seat lookup, booking creation, seat selection`);
}