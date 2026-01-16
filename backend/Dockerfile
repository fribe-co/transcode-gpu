# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy source code first (for go mod tidy)
COPY . .

# Download dependencies and tidy
RUN go mod tidy && go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Final stage
FROM ubuntu:22.04

WORKDIR /app

# Prevent interactive prompts during apt install
ENV DEBIAN_FRONTEND=noninteractive

# Install runtime dependencies
# ffmpeg in Ubuntu 22.04 has NVENC support out of the box
RUN apt-get update && apt-get install -y \
    ca-certificates \
    ffmpeg \
    numactl \
    && rm -rf /var/lib/apt/lists/*

# Copy binary from builder
COPY --from=builder /app/main .
COPY --from=builder /app/migrations ./migrations

# Create directories for storage
RUN mkdir -p /var/lib/cashbacktv/streams /var/lib/cashbacktv/logos /var/lib/cashbacktv/uploads

# Copy entrypoint script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Expose port
EXPOSE 8080

# Use entrypoint script to set up nvidia-smi symlink
ENTRYPOINT ["/entrypoint.sh"]

# Run the application
CMD ["./main"]

