# Build stage
FROM golang:1.27.3-alpine3.23@sha256:8a5046497cda39860b832ffdbe8dd66018a0f05bb01352f533a36e7e2ced66a0 AS builder

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
    -o apibconv ./cmd/apibconv

# Runtime stage
FROM scratch

# Copy CA certificates for HTTPS support
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /build/apibconv /apibconv
COPY --from=builder /build/LICENSE /LICENSE
COPY --from=builder /build/THIRD_PARTY_NOTICES.md /THIRD_PARTY_NOTICES.md

# Use a non-root user (ID 65532 is commonly used for nonroot)
USER 65532:65532

WORKDIR /data

ENTRYPOINT ["/apibconv"]
CMD ["--help"]
