# Шаг для компиляции backend
FROM golang:1.24-bookworm AS backend_builder

ARG BACKEND_PATH=.
ARG CMD=api

WORKDIR /app
RUN go env -w GOMODCACHE=/root/.cache/go-build

COPY $BACKEND_PATH/go.mod .
COPY $BACKEND_PATH/go.sum .
RUN --mount=type=cache,target=/root/.cache/go-build \
    go mod download

WORKDIR /app
COPY $BACKEND_PATH/cmd ./cmd
COPY $BACKEND_PATH/internal ./internal
COPY $BACKEND_PATH/pkg ./pkg

WORKDIR /app/cmd/$CMD
RUN --mount=type=cache,target="/root/.cache/go-build" \
    go build -tags "sqlite_fts5" -ldflags="-s -w" -o main .

# Основной шаг
FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y \
    curl

ARG BACKEND_PATH=.
ARG CMD=api

WORKDIR /app

COPY $BACKEND_PATH/migrations ./.migrations
COPY --from=backend_builder /app/cmd/$CMD/main .
COPY ./migrations ./migrations
CMD ["./main", "-m", "./migrations"]
