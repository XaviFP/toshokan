# =============================================================================
# Stage 1: Build
# =============================================================================
FROM rust:latest AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    protobuf-compiler \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy manifests for dependency caching
COPY Cargo.toml Cargo.lock* ./

# Copy build script (needed for tonic-build)
COPY build.rs ./

# Copy proto file (needed for build.rs)
COPY api/proto/v1/dealer.proto api/proto/v1/

# Create dummy main to cache dependencies
RUN mkdir src && \
    echo "fn main() {}" > src/main.rs && \
    cargo build --release && \
    rm -rf src

# Copy actual source code
COPY src/ src/

# Touch main.rs to invalidate the cached dummy build
RUN touch src/main.rs

# Build the release binary
RUN cargo build --release

# =============================================================================
# Stage 2: Runtime
# =============================================================================
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    netcat-openbsd \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -u 1001 appuser

COPY --from=builder --chmod=755 /app/target/release/dealer /dealer

USER appuser

EXPOSE 50051
ENTRYPOINT ["/dealer"]
