# The building
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binary
RUN go build -o server .

# Run Binary
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/server .
# Preserve the db directory because the application reads ./db/*.sql.
COPY --from=builder /app/db ./db

EXPOSE 8080

CMD ["./server"]
