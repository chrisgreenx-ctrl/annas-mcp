# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o annas-mcp ./cmd/annas-mcp

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/annas-mcp .

# Create downloads directory
RUN mkdir -p /tmp/downloads

# Set default environment variables (can be overridden)
ENV ANNAS_SECRET_KEY="" \
    ANNAS_DOWNLOAD_PATH="/tmp/downloads" \
    PORT="8080"

# Expose the default HTTP port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the HTTP server by default with SSE transport (better Smithery compatibility)
CMD ["./annas-mcp", "http", "--host", "0.0.0.0", "--port", "8080", "--transport", "sse"]
