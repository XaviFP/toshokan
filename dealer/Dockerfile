# Build stage
FROM rust:1.85-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    protobuf-compiler \
    libprotobuf-dev \
    && rm -rf /var/lib/apt/lists/*

# Built from the repo root, so build.rs can read the course service's proto.
WORKDIR /app/dealer

COPY dealer/Cargo.toml dealer/Cargo.lock* ./

COPY dealer/build.rs ./

COPY dealer/api/proto/v1/dealer.proto api/proto/v1/

COPY course/api/proto/v1/course.proto /app/course/api/proto/v1/

# Create dummy main to cache dependencies
RUN mkdir src && \
    echo "fn main() {}" > src/main.rs && \
    cargo build --release && \
    rm -rf src

COPY dealer/src/ src/

# Touch main.rs to invalidate the cached dummy build
RUN touch src/main.rs

RUN cargo build --release

# Runtime stage
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    netcat-openbsd \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -u 1001 appuser

COPY --from=builder --chmod=755 /app/dealer/target/release/dealer /dealer

USER appuser

EXPOSE 50051
ENTRYPOINT ["/dealer"]
