# Dockerfile for Forge
# Multi-stage build for smaller final image

# Stage 1: Builder
FROM --platform=$BUILDPLATFORM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build arguments
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

# Build the application
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w \
        -X github.com/kkkldpz/forge/internal/version.Version=${VERSION} \
        -X github.com/kkkldpz/forge/internal/version.Commit=${COMMIT} \
        -X github.com/kkkldpz/forge/internal/version.Date=${DATE}" \
    -o /tmp/forge

# Stage 2: Final image
FROM alpine:3.19

# Labels
LABEL maintainer="Forge Team <forge@example.com>"
LABEL description="AI-powered CLI assistant"
LABEL version="1.0"
LABEL org.opencontainers.image.title="Forge"
LABEL org.opencontainers.image.description="AI-powered CLI assistant"
LABEL org.opencontainers.image.source="https://github.com/kkkldpz/forge"

# Install runtime dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    bash \
    ripgrep

# Create non-root user
RUN addgroup -g 1000 forge && \
    adduser -u 1000 -G forge -s /bin/bash -D forge

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /tmp/forge /app/forge

# Copy configuration directory structure
RUN mkdir -p /app/.forge /app/.config/forge && \
    chown -R forge:forge /app

# Copy entrypoint script
COPY <<'EOF' /app/entrypoint.sh
#!/bin/bash
set -e

# If no API key provided via env, check for config
if [ -z "$ANTHROPIC_API_KEY" ]; then
    if [ -f "$HOME/.forge/api_key" ]; then
        export ANTHROPIC_API_KEY=$(cat "$HOME/.forge/api_key")
    fi
fi

exec /app/forge "$@"
EOF

RUN chmod +x /app/entrypoint.sh

# Switch to non-root user
USER forge

# Environment variables
ENV HOME=/app
ENV PATH=/app:$PATH
ENV FORGE_CONFIG_DIR=/app/.forge
ENV FORGE_DATA_DIR=/app/.config/forge

# Default volume for config persistence
VOLUME ["/app/.forge", "/app/.config/forge"]

# Default command
ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["chat"]

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD pgrep forge > /dev/null || exit 1

# Expose port (for server mode)
EXPOSE 18789