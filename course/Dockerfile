# =============================================================================
# Stage 1: Proto Generation
# =============================================================================
# When built with docker-bake, uses shared proto-builder target.
# When built standalone, generates protos locally.
# =============================================================================
FROM golang:1.24-bookworm AS proto-builder-local

RUN apt-get update && apt-get install -y --no-install-recommends \
    protobuf-compiler \
    libprotobuf-dev \
    && rm -rf /var/lib/apt/lists/*

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

WORKDIR /proto

# Course depends on deck proto
COPY deck/api/proto/v1/deck.proto deck/api/proto/v1/
COPY course/api/proto/v1/course.proto course/api/proto/v1/

RUN protoc --proto_path=deck/api/proto/v1/ \
    --go_out=deck/api/proto/v1/ --go_opt=paths=source_relative \
    --go-grpc_out=deck/api/proto/v1/ --go-grpc_opt=paths=source_relative \
    --go-grpc_opt=require_unimplemented_servers=false \
    deck.proto

RUN protoc --proto_path=course/api/proto/v1/ \
    --proto_path=/usr/include \
    --go_out=course/api/proto/v1/ --go_opt=paths=source_relative \
    --go-grpc_out=course/api/proto/v1/ --go-grpc_opt=paths=source_relative \
    --go-grpc_opt=require_unimplemented_servers=false \
    course.proto

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
COPY course/ course/

# Copy deck (needed for proto imports)
COPY deck/ deck/

# Copy generated proto files from proto-builder
COPY --from=proto-builder /proto/course/api/proto/v1/*.pb.go course/api/proto/v1/
COPY --from=proto-builder /proto/deck/api/proto/v1/*.pb.go deck/api/proto/v1/

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /course ./course/cmd/course/

# =============================================================================
# Stage 3: Runtime
# =============================================================================
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    netcat-openbsd \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -u 1001 appuser

COPY --from=builder --chmod=755 /course /course

USER appuser

EXPOSE 50051
ENTRYPOINT ["/course"]
