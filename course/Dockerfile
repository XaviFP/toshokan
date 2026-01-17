FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    netcat-openbsd \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -u 1001 appuser

COPY --chmod=755 bin/course /course

USER appuser

EXPOSE 50051
ENTRYPOINT ["/course"]
