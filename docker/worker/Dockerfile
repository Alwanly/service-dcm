# Build stage
FROM golang:1.25.6-alpine3.23 AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Install swag for generating swagger docs
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Copy source code
COPY . .

# Generate swagger documentation
RUN swag init -g cmd/worker/main.go -o docs/worker

# Build worker (no CGO needed)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o worker ./cmd/worker

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/worker .

# Run as non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

# Set default environment variables
ENV WORKER_ADDR=:8082 \
    REQUEST_TIMEOUT=10

# Expose worker port
EXPOSE 8082

ENTRYPOINT ["/app/worker"]
