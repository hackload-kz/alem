#!/usr/bin/env bash
set -euo pipefail

# HOST="https://sunny-reasonably-shepherd.ngrok-free.app"
# HOST="http://82.115.42.43"
# HOST="http://alem.hub.hackload.kz"
HOST="http://localhost:8080"

ab -n 1000 -c 50 -t 30 \
   -H "Authorization: Basic YXlzdWx0YW5fdGFsZ2F0XzFAZmVzdC50aXg6LzhlQyRBRD4=" \
   -r \
   "$HOST/api/events?query=концерт&date=2024-12-25&page=1&pageSize=20" # <- this is killing
