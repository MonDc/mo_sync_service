# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o mo_sync_service ./cmd/api

# Run stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/mo_sync_service .
COPY --from=builder /app/config/config.yaml ./config/
COPY --from=builder /app/scripts/migrations ./scripts/migrations

EXPOSE 8081 9090

CMD ["./mo_sync_service"]
