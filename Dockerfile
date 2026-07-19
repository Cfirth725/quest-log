# ====================================================================
# -- STAGE 1: COMPILATION & ARCHITECTURE BUILDER --
# ====================================================================
FROM golang:1.24-bookworm AS builder

# Enforce CGO compilation to ensure the mattn/go-sqlite3 C-bindings link correctly
ENV CGO_ENABLED=1

# Install essential compilation toolchains for SQLite C-bindings
RUN apt-get update && apt-get install -y \
    gcc \
    libc6-dev \
    libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Cache Layer optimization: Copy and download modules first
COPY go.mod go.sum ./
RUN go mod download

# Ingest all sub-packages and directories required for compilation
COPY internal/ ./internal/
COPY cmd/ ./cmd/

# Build the binary targeting your new cmd/main.go entrypoint location
RUN go build -o quest-log ./cmd/main.go

# ====================================================================
# -- STAGE 2: IMMUTABLE DISTROLITH RUNTIME ENVIRONMENT --
# ====================================================================
FROM debian:bookworm-slim

# Install light dynamic runtime libraries, health check dependencies, and tzdata
RUN apt-get update && apt-get install -y \
    libsqlite3-0 \
    wget \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Secure and pull the compiled Go runtime binary from the builder stage
COPY --from=builder /app/quest-log .

# Ingest frontend visualization assets and document layouts
COPY templates/ ./templates/
COPY static/ ./static/

# Establish a persistent data folder block on the container host filesystem
RUN mkdir -p /app/data && chmod -R 755 /app

# Document the designated deployment communication socket
EXPOSE 8081

# Expose the data path as a volume mount location for long-term database persistence
VOLUME ["/app/data"]

CMD ["./quest-log"]