# Build stage
FROM golang:1.25.6-alpine3.23 AS builder

# Install build dependencies for CGO and SQLite
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Install swag for generating swagger docs
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Copy source code
COPY . .

# Generate swagger documentation
RUN swag init -g cmd/controller/main.go -o docs/controller

# Build with CGO enabled for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o controller ./cmd/controller

# Runtime stage
FROM alpine:latest

# Install runtime dependencies (include sqlite CLI for init script)
RUN apk add --no-cache ca-certificates sqlite-libs sqlite

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/controller .


# Create data directory for SQLite database
RUN mkdir -p /app/data

# Expose controller port
EXPOSE 8080


# Run as non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

# Set default environment variables
ENV CONTROLLER_ADDR=:8080 \
    DATABASE_PATH=/app/data/data.db \
    POLL_INTERVAL=5


ENTRYPOINT ["/app/controller"]
