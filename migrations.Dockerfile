# =============================================================================
# Migrations Runner
# =============================================================================
# Runs database migrations for a specified service
#
# Usage:
#   docker build -f migrations.Dockerfile -t toshokan-migrations .
#   docker run --rm --network toshokan \
#     -e DB_HOST=db -e DB_PORT=5432 -e DB_USER=toshokan -e DB_PASSWORD=t.o.s.h.o.k.a.n. \
#     -e SERVICE=deck -e DB_NAME=deck \
#     toshokan-migrations
#
# SERVICE can be: deck, user, course, all
# =============================================================================

FROM golang:1.24-bookworm AS builder

WORKDIR /app

# Copy go.mod and go.sum from root for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy common packages
COPY common/ common/

# Copy all services that have migrations
COPY deck/ deck/
COPY user/ user/
COPY course/ course/

# Build migration runners for each service
RUN CGO_ENABLED=0 GOOS=linux go build -o /migrate-deck ./deck/cmd/migrate/
RUN CGO_ENABLED=0 GOOS=linux go build -o /migrate-user ./user/cmd/migrate/
RUN CGO_ENABLED=0 GOOS=linux go build -o /migrate-course ./course/cmd/migrate/

# =============================================================================
# Runtime
# =============================================================================
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    netcat-openbsd \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /migrate-deck /migrate-deck
COPY --from=builder /migrate-user /migrate-user
COPY --from=builder /migrate-course /migrate-course

# Copy migration files
COPY deck/cmd/migrate/migrations/ /migrations/deck/
COPY user/cmd/migrate/migrations/ /migrations/user/
COPY course/cmd/migrate/migrations/ /migrations/course/

# Copy entrypoint script
COPY scripts/migrate-entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
