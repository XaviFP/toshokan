# =============================================================================
# Stage 1: Proto Generation
# =============================================================================
# When built with docker-bake, uses shared proto-builder target.
# When built standalone, generates protos locally.
# =============================================================================
FROM golang:1.24-bookworm AS proto-builder-local

RUN apt-get update && apt-get install -y --no-install-recommends \
    protobuf-compiler \
    && rm -rf /var/lib/apt/lists/*

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

WORKDIR /proto

COPY deck/api/proto/v1/deck.proto deck/api/proto/v1/

RUN protoc --proto_path=deck/api/proto/v1/ \
    --go_out=deck/api/proto/v1/ --go_opt=paths=source_relative \
    --go-grpc_out=deck/api/proto/v1/ --go-grpc_opt=paths=source_relative \
    --go-grpc_opt=require_unimplemented_servers=false \
    deck.proto

# Use external proto-builder if provided, otherwise use local
FROM proto-builder-local AS proto-builder

# =============================================================================
# Stage 2: Build
# =============================================================================
FROM golang:1.24-bookworm AS builder

WORKDIR /app

# Copy go.mod and go.sum from root for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy common packages
COPY common/ common/

# Copy service code
COPY deck/ deck/

# Copy generated proto files from proto-builder
COPY --from=proto-builder /proto/deck/api/proto/v1/*.pb.go deck/api/proto/v1/

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /deck ./deck/cmd/deck/

# =============================================================================
# Stage 3: Runtime
# =============================================================================
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    netcat-openbsd \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -u 1001 appuser

COPY --from=builder --chmod=755 /deck /deck

USER appuser

EXPOSE 50051
ENTRYPOINT ["/deck"]
