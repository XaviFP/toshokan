# =============================================================================
# Go Services
# =============================================================================
# One target per service: deck, user, course, gate.
# Builds from the committed .pb.go files; `make proto` regenerates them.
#
# Build: docker build --target gate -t toshokan-gate .
# =============================================================================

FROM golang:1.27-bookworm AS builder

WORKDIR /app

# Copy go.mod and go.sum from root for dependency caching
COPY go.mod go.sum ./
RUN go mod download

COPY common/ common/
COPY deck/ deck/
COPY user/ user/
COPY course/ course/
COPY gate/ gate/
COPY dealer/api/proto/v1/*.go dealer/api/proto/v1/

FROM builder AS deck-builder
RUN CGO_ENABLED=0 GOOS=linux go build -o /deck ./deck/cmd/deck/

FROM builder AS user-builder
RUN CGO_ENABLED=0 GOOS=linux go build -o /user ./user/cmd/user/

FROM builder AS course-builder
RUN CGO_ENABLED=0 GOOS=linux go build -o /course ./course/cmd/course/

FROM builder AS gate-builder
RUN CGO_ENABLED=0 GOOS=linux go build -o /gate ./gate/cmd/gate/

# protoc and its Go plugins, which `make proto` runs
FROM golang:1.27-bookworm AS protoc

RUN apt-get update && apt-get install -y --no-install-recommends \
    protobuf-compiler \
    libprotobuf-dev \
    && rm -rf /var/lib/apt/lists/*

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.1

# =============================================================================
# Runtime
# =============================================================================
FROM debian:bookworm-slim AS runtime

RUN apt-get update && apt-get install -y --no-install-recommends \
    netcat-openbsd \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -u 1001 appuser

USER appuser

FROM runtime AS deck
COPY --from=deck-builder --chmod=755 /deck /deck
EXPOSE 50051
ENTRYPOINT ["/deck"]

FROM runtime AS user
COPY --from=user-builder --chmod=755 /user /user
EXPOSE 50051
ENTRYPOINT ["/user"]

FROM runtime AS course
COPY --from=course-builder --chmod=755 /course /course
EXPOSE 50051
ENTRYPOINT ["/course"]

FROM runtime AS gate
COPY --from=gate-builder --chmod=755 /gate /gate
EXPOSE 8080
ENTRYPOINT ["/gate"]
