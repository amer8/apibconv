# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o apibconv .

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/apibconv /usr/local/bin/apibconv

# Create data directory
RUN mkdir /data
WORKDIR /data

# Run as non-root user
RUN adduser -D -u 1000 apibconv
USER apibconv

ENTRYPOINT ["apibconv"]
CMD ["--help"]
