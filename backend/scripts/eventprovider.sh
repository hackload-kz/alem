#!/usr/bin/env bash
set -euo pipefail

ADDR="https://hub.hackload.kz/event/736254/event-provider"

ORDER_ID="032fd6d4-eefa-4c94-ba2b-0b246589f3f7"
PLACE_ID="343fe140-6cc9-4783-b2f7-807ba7f2606b"

# Create Order
# curl -X POST "$ADDR/api/partners/v1/orders"

# Get Places
# curl "$ADDR/api/partners/v1/places?page=1&pageSize=5"
# curl "$ADDR/api/partners/v1/places/$PLACE_ID"

# Get Order
# curl "$ADDR/api/partners/v1/orders/$ORDER_ID"

# Book Place
# curl -vvv -X PATCH "$ADDR/api/partners/v1/places/$PLACE_ID/select" \
#   -H "Content-Type: application/json" \
#   -d "{\"order_id\": \"$ORDER_ID\"}"

# Release Place
# curl -vvv -X PATCH "$ADDR/api/partners/v1/places/$PLACE_ID/release"

# Submit Order
# curl -vvv -X PATCH "$ADDR/api/partners/v1/orders/$ORDER_ID/submit"

# Cancel Order
# curl -vvv -X PATCH "$ADDR/api/partners/v1/orders/$ORDER_ID/cancel"
