# Build stage
FROM golang:1.24.4-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build agent (no CGO needed)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o agent ./cmd/agent

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/agent .

# Run as non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

# Set default environment variables
ENV CONTROLLER_URL=http://controller:8080 \
    WORKER_URL=http://worker:8082 \
    POLL_INTERVAL=5 \
    REQUEST_TIMEOUT=10

ENTRYPOINT ["/app/agent"]
