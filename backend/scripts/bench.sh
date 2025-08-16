#!/usr/bin/env bash
set -euo pipefail

ab -n 10 \
   -H "Authorization: Basic YXlzdWx0YW5fdGFsZ2F0XzFAZmVzdC50aXg6LzhlQyRBRD4=" \
   -r \
   "https://hub.hackload.kz/event/alem/event-provider/api/partners/v1/places"
