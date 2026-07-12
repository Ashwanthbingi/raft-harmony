# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o raft-node ./cmd/node/main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /build/raft-node .

# Create data directory for BoltDB
RUN mkdir -p /app/data

# Expose ports
# 8000: HTTP API (configurable per node)
# 9000: Raft gRPC (configurable per node)
EXPOSE 8000 9000

# Health check
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8000/health || exit 1

# Set environment variables
ENV LOG_LEVEL=INFO
ENV DATA_DIR=/app/data

# Default command (can be overridden)
ENTRYPOINT ["./raft-node"]
CMD ["-id", "1", "-http_addr", ":8000", "-raft_addr", ":9000"]
