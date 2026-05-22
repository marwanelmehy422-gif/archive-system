# ── Stage 1: Build ──────────────────────────────────────────
FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o archive-server ./cmd/server

# ── Stage 2: Run ────────────────────────────────────────────
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/archive-server .

RUN mkdir -p /app/uploads/transactions

EXPOSE 8080

CMD ["./archive-server"]