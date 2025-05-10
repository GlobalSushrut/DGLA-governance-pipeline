# Stage 1: Build the application
FROM golang:1.18-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata build-base

WORKDIR /app

# Copy go mod files first for better layer caching
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application with proper flags for production
ARG VERSION=1.0.0
ARG BUILD_DATE
ARG COMMIT_HASH

# Build with version info and security optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a \
    -ldflags "-w -s \
    -X main.serviceVersion=${VERSION} \
    -X main.buildDate=${BUILD_DATE} \
    -X main.commitHash=${COMMIT_HASH} \
    -extldflags \"-static\"" \
    -o dgla

# Run tests
RUN go test -v ./...

# Stage 2: Runtime image using distroless for security
FROM gcr.io/distroless/static:nonroot AS runtime

# Define arguments and labels
ARG VERSION=1.0.0
ARG BUILD_DATE
ARG COMMIT_HASH

# Add metadata labels
LABEL org.opencontainers.image.title="DGLA Data Governance Pipeline"
LABEL org.opencontainers.image.description="Production-grade data governance and lineage tracking system"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.created="${BUILD_DATE}"
LABEL org.opencontainers.image.revision="${COMMIT_HASH}"
LABEL org.opencontainers.image.vendor="DGLA"

WORKDIR /app

# Copy the binary and configs from the builder stage
COPY --from=builder /app/dgla .
COPY --from=builder /app/config.json .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Create directory structure
COPY --from=builder --chown=nonroot:nonroot /app/agreements /app/agreements

# Set up environment variables with safe defaults
ENV DGLA_SERVER_PORT=8081
ENV DGLA_LOG_LEVEL=info
ENV DGLA_AUTH_ENABLED=true
ENV DGLA_CACHE_TYPE=memory
ENV TZ=UTC

# Use nonroot user for security (already set in distroless image)
# UID 65532 is the nonroot user in the distroless image

# Expose the port
EXPOSE 8081

# Enhanced health checks
# Note: We use curl from the slim image because distroless doesn't have wget
COPY --from=alpine:latest /usr/bin/wget /usr/bin/wget
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/alive || exit 1

# Define volumes for persistent data
VOLUME ["/app/logs"]

# Set read-only filesystem except for volumes
RUN chmod 555 /app/dgla

# Run the application with configuration
ENTRYPOINT ["/app/dgla"]
CMD ["--config", "config.json"]
