# Build stage
FROM golang:1.23-alpine AS builder

# Install git and ca-certificates (needed for fetching dependencies)
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/bin/ask-moto \
    ./cmd/server

# Runtime stage
FROM alpine:3.19

# Install ca-certificates for HTTPS requests (if needed in future)
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user for security
RUN addgroup -S askmoto && adduser -S askmoto -G askmoto

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/ask-moto /app/ask-moto

# Copy knowledge base data
COPY --from=builder /app/data/kb /app/data/kb

# Create logs directory
RUN mkdir -p /app/data/logs && chown -R askmoto:askmoto /app

# Switch to non-root user
USER askmoto

# Expose port
EXPOSE 8080

# Set default environment variables
ENV PORT=8080 \
    KB_PATH=/app/data/kb \
    KB_VERSION=v1 \
    LOGS_PATH=/app/data/logs \
    TOP_K=5 \
    MIN_RETRIEVAL_SCORE=1.0 \
    INTENT_MIN_CONFIDENCE=0.3 \
    INTENT_MATCH_THRESHOLD=0.7 \
    SAFE_DEFAULT_CONFIDENCE=0.7 \
    QUESTION_OVERLAP_THRESHOLD=0.7 \
    SENTENCE_RELEVANCE_THRESHOLD=0.4 \
    KEYWORD_BOOST_HIGH=3.0 \
    KEYWORD_BOOST_MEDIUM=2.5 \
    KEYWORD_BOOST_LOW=1.5

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT}/health || exit 1

# Run the application
ENTRYPOINT ["/app/ask-moto"]
