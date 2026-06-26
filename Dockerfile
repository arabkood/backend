FROM --platform=$BUILDPLATFORM golang:1.24.0-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with production optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/main ./cmd/server

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy binary from build stage
COPY --from=builder /app/main /app/main
# COPY --from=builder /app/config/config.prod.yaml /app/config.yaml

# Create a non-root user to run the application
RUN adduser -D appuser
USER appuser

# Expose the application port
EXPOSE 8080

# Command to run the application
CMD ["/app/main"]
