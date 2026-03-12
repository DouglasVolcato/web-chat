# Build stage
FROM golang:1.24-alpine AS builder

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the binary
# CGO_ENABLED=0 is used for a static binary that works on alpine
RUN CGO_ENABLED=0 GOOS=linux go build -o binaryApp .

# Run stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/binaryApp .

# Copy environment example if needed, but usually passed via env vars
# COPY .env .

# Copy folders required at runtime
COPY --from=builder /app/presentation ./presentation
COPY --from=builder /app/migrations ./migrations

# Expose port (default for most apps, adjust if needed)
EXPOSE 8080

# Run the binary
CMD ["./binaryApp"]
