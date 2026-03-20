FROM golang:1.25-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -o chperf .

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/chperf .
COPY --from=builder /app/web/static ./web/static

ENV SQLITE_PATH=/app/data/chperf.db

EXPOSE 8085

CMD ["./chperf"]
