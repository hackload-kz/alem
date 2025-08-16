#!/usr/bin/env bash
set -euo pipefail

ab -n 1000 -c 50 -t 30 \
   -H "Authorization: Basic YXlzdWx0YW5fdGFsZ2F0XzFAZmVzdC50aXg6LzhlQyRBRD4=" \
   -r \
   "http://localhost:8080/api/events"
