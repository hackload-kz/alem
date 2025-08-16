#!/usr/bin/env bash
set -euo pipefail


API_ADDR="http://localhost:8080"
AUTH_HEADER="Authorization: Basic YXlzdWx0YW5fdGFsZ2F0XzFAZmVzdC50aXg6LzhlQyRBRD4="

curl -H "$AUTH_HEADER" "$API_ADDR/api/events?query=Иван"
# curl -H "$AUTH_HEADER" "$API_ADDR/api/bookings"
# curl -H "$AUTH_HEADER" "$API_ADDR/api/seats?event_id=1"
# curl -vvv -H "$AUTH_HEADER" -X PATCH -d '{"seat_id": 1}' "$API_ADDR/api/seats/release"
# curl -vvv -H "$AUTH_HEADER" -X PATCH -d '{"seat_id": 1, "booking_id": 1}' "$API_ADDR/api/seats/select"
