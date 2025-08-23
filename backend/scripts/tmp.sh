#!/usr/bin/env bash
set -euo pipefail


# API_ADDR="https://alem.hub.hackload.kz"
API_ADDR="http://localhost:8080"
AUTH_HEADER="Authorization: Basic YXlzdWx0YW5fdGFsZ2F0XzFAZmVzdC50aXg6LzhlQyRBRD4="

BOOKING_ID="5"
SEAT_ID="200002"

set -x

# curl -s -H "$AUTH_HEADER" -X POST "$API_ADDR/api/reset"
# curl  -s -H "$AUTH_HEADER" "$API_ADDR/api/analytics?id=1"
# curl  -s -H "$AUTH_HEADER" "$API_ADDR/api/events"
# curl  -s -H "$AUTH_HEADER" "$API_ADDR/api/bookings"
# curl  -s -H "$AUTH_HEADER" -X POST -d '{"event_id": 1}' "$API_ADDR/api/bookings"
# curl  -s -H "$AUTH_HEADER" "$API_ADDR/api/seats?event_id=1"
# curl  -s -vvv -H "$AUTH_HEADER" -X PATCH -d "{\"seat_id\":$SEAT_ID, \"booking_id\":$BOOKING_ID}" "$API_ADDR/api/seats/select"
# curl  -s -vvv -H "$AUTH_HEADER" -X PATCH -d "{\"seat_id\":$SEAT_ID}" "$API_ADDR/api/seats/release"
# curl  -s -vvv -H "$AUTH_HEADER" -X PATCH -d "{\"booking_id\":$BOOKING_ID}" "$API_ADDR/api/bookings/initiatePayment"

# curl -X GET "https://sunny-reasonably-shepherd.ngrok-free.app/api/payments/success?orderId=1755943241054941000"
