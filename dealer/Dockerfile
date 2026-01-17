FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    netcat-openbsd \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -u 1001 appuser

# Use release build for production
COPY --chmod=755 target/release/dealer /dealer

USER appuser

EXPOSE 50051
ENTRYPOINT ["/dealer"]
