# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build arguments
ARG VERSION=dev

# Build static binary
# -ldflags="-s -w" strips debug symbols for smaller size
# -trimpath removes file system paths from the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o apibconv .

# Runtime stage
FROM scratch

# Copy CA certificates for HTTPS support
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /build/apibconv /apibconv

# Use a non-root user (ID 65532 is commonly used for nonroot)
USER 65532:65532

WORKDIR /data

ENTRYPOINT ["/apibconv"]
CMD ["--help"]
