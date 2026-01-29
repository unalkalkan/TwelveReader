# TwelveReader Multi-stage Dockerfile
# Stage 1: Build Go backend
FROM golang:1.24-alpine AS backend-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o twelvereader ./cmd/server

# Stage 2: Build frontend
FROM node:22-alpine AS frontend-builder

WORKDIR /app

# Copy package files first for better caching
COPY web-client/package.json web-client/package-lock.json* ./

# Install dependencies
RUN npm install

# Copy source code
COPY web-client/ ./

# Build the frontend
RUN npm run build

# Stage 3: Final runtime image
FROM alpine:3.21

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' appuser

# Create directories for data and config
RUN mkdir -p /app/data /app/config /app/static && \
    chown -R appuser:appuser /app

# Copy binary from backend builder
COPY --from=backend-builder /app/twelvereader .

# Copy frontend build from frontend builder
COPY --from=frontend-builder /app/dist ./static/

# Copy example config
COPY config/dev.example.yaml ./config/

# Set ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default command
ENTRYPOINT ["./twelvereader"]
CMD ["-config", "/app/config/config.yaml"]
