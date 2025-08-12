# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o monitor ./cmd/monitor

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS connections
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S ethmonitor && \
    adduser -u 1001 -S ethmonitor -G ethmonitor

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/monitor .


# Change ownership to non-root user
RUN chown -R ethmonitor:ethmonitor /app

# Switch to non-root user
USER ethmonitor

# Expose port (if needed for health checks)
EXPOSE 8080

# Command to run the application
CMD ["./monitor", "-conf", "config.yaml"]