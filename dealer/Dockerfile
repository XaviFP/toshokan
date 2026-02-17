# Build stage
FROM rust:1.83-bookworm AS builder

WORKDIR /app
COPY Cargo.toml build.rs ./
COPY src ./src

RUN cargo build --release

# Runtime stage
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    netcat-openbsd \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -u 1001 appuser

COPY --from=builder --chmod=755 /app/target/release/dealer /dealer

USER appuser

EXPOSE 50051
ENTRYPOINT ["/dealer"]
