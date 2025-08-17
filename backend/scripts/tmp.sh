#!/usr/bin/env bash
set -euo pipefail


# API_ADDR="https://alem.hub.hackload.kz"
API_ADDR="http://localhost:8080"
AUTH_HEADER="Authorization: Basic YXlzdWx0YW5fdGFsZ2F0XzFAZmVzdC50aXg6LzhlQyRBRD4="

set -x

# curl -H "$AUTH_HEADER" -X POST "$API_ADDR/api/reset"
curl -H "$AUTH_HEADER" "$API_ADDR/api/analytics?id=1"
# curl -H "$AUTH_HEADER" "$API_ADDR/api/events"
# curl -H "$AUTH_HEADER" "$API_ADDR/api/bookings"
# curl -H "$AUTH_HEADER" -X POST -d '{"event_id": 1}' "$API_ADDR/api/bookings"
# curl -H "$AUTH_HEADER" "$API_ADDR/api/seats?event_id=1"
# curl -vvv -H "$AUTH_HEADER" -X PATCH -d '{"seat_id": 2, "booking_id": 2}' "$API_ADDR/api/seats/select"
# curl -vvv -H "$AUTH_HEADER" -X PATCH -d '{"seat_id": 1}' "$API_ADDR/api/seats/release"
# curl -vvv -H "$AUTH_HEADER" -X PATCH -d '{"booking_id": 2}' "$API_ADDR/api/bookings/initiatePayment"
