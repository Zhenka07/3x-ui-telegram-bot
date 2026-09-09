# Stage 1: Build binary
FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /build/admin-bot ./cmd/admin

# Stage 2: Minimal runtime
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/admin-bot /app/admin-bot

RUN mkdir -p /app/data && chmod 700 /app/data

VOLUME ["/app/data"]

ENTRYPOINT ["/app/admin-bot"]
