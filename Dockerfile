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

# Expose the default HTTP port
EXPOSE 8080

# Run the HTTP server by default
CMD ["./annas-mcp", "http", "--host", "0.0.0.0", "--port", "8080", "--transport", "streamable"]
