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
COPY . .
# This builds the current directory and all subdirectories
RUN go build -o quest-log .

# --- Final Stage ---
FROM debian:bookworm-slim

# Install runtime SQLite library AND wget for healthchecks
RUN apt-get update && apt-get install -y \
    libsqlite3-0 \
    wget \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/quest-log .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static

RUN chmod +x /app/quest-log

CMD ["./quest-log"]