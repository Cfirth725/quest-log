# We use 'bookworm' instead of 'alpine' for better SQLite compatibility
FROM golang:1.24-bookworm AS builder

# 1. Force CGO on
ENV CGO_ENABLED=1

# 2. Install standard build tools
RUN apt-get update && apt-get install -y \
    gcc \
    libc6-dev \
    libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# 3. Handle dependencies
COPY go.mod go.sum ./
RUN go mod download

# 4. Build
COPY internal/ ./internal/
COPY main.go ./main.go
# This builds the current directory and all subdirectories
RUN go build -o quest-log .

# --- Final Stage ---
FROM debian:bookworm-slim

# Install runtime SQLite library, wget, and the timezone database
RUN apt-get update && apt-get install -y \
    libsqlite3-0 \
    wget \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# 1. Pull the compiled binary from the builder stage
COPY --from=builder /app/quest-log .

# 2. Pull the static folders directly from your local host context
COPY templates ./templates
COPY static ./static

RUN chmod +rw /app

CMD ["./quest-log"]