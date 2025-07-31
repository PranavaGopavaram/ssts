# Build stage for Go application
FROM golang:1.21-alpine AS go-builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod ./

# Copy source code to resolve dependencies
COPY . .

# Initialize go.sum and download dependencies
RUN go mod tidy && go mod download

# Create a simple web dist directory for now
RUN mkdir -p web/dist && echo '<html><body><h1>SSTS API Server</h1><p>Web UI coming soon...</p></body></html>' > web/dist/index.html

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags="-w -s -X main.version=1.0.0 -X main.commit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown') -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o ssts ./cmd/ssts

# Final stage - minimal runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1001 -S ssts && \
    adduser -u 1001 -S ssts -G ssts

WORKDIR /app

# Copy the binary
COPY --from=go-builder /app/ssts .

# Copy configuration files
COPY --from=go-builder /app/ssts.yaml .
COPY --from=go-builder /app/examples ./examples

# Copy web assets
COPY --from=go-builder /app/web/dist ./web/dist

# Create data directory and set permissions
RUN mkdir -p /data && \
    chown -R ssts:ssts /app /data

# Switch to non-root user
USER ssts

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run the application
CMD ["./ssts", "server"]