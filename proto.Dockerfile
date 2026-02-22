# =============================================================================
# Proto Builder
# =============================================================================
# Generates all proto files for the project in one place.
# Used as a dependency by all service Dockerfiles.
#
# Build: docker build -f proto.Dockerfile -t toshokan-proto .
# =============================================================================

FROM golang:1.24-bookworm

RUN apt-get update && apt-get install -y --no-install-recommends \
    protobuf-compiler \
    libprotobuf-dev \
    && rm -rf /var/lib/apt/lists/*

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

WORKDIR /proto

# Copy all proto files
COPY deck/api/proto/v1/deck.proto deck/api/proto/v1/
COPY user/api/proto/v1/user.proto user/api/proto/v1/
COPY course/api/proto/v1/course.proto course/api/proto/v1/
COPY dealer/api/proto/v1/dealer.proto dealer/api/proto/v1/

# Generate Go code for all protos
RUN protoc --proto_path=deck/api/proto/v1/ \
    --go_out=deck/api/proto/v1/ --go_opt=paths=source_relative \
    --go-grpc_out=deck/api/proto/v1/ --go-grpc_opt=paths=source_relative \
    --go-grpc_opt=require_unimplemented_servers=false \
    deck.proto

RUN protoc --proto_path=user/api/proto/v1/ \
    --go_out=user/api/proto/v1/ --go_opt=paths=source_relative \
    --go-grpc_out=user/api/proto/v1/ --go-grpc_opt=paths=source_relative \
    --go-grpc_opt=require_unimplemented_servers=false \
    user.proto

RUN protoc --proto_path=course/api/proto/v1/ \
    --proto_path=/usr/include \
    --go_out=course/api/proto/v1/ --go_opt=paths=source_relative \
    --go-grpc_out=course/api/proto/v1/ --go-grpc_opt=paths=source_relative \
    --go-grpc_opt=require_unimplemented_servers=false \
    course.proto

RUN protoc --proto_path=dealer/api/proto/v1/ \
    --go_out=dealer/api/proto/v1/ --go_opt=paths=source_relative \
    --go-grpc_out=dealer/api/proto/v1/ --go-grpc_opt=paths=source_relative \
    --go-grpc_opt=require_unimplemented_servers=false \
    dealer.proto

# Output structure:
# /proto/deck/api/proto/v1/*.pb.go
# /proto/user/api/proto/v1/*.pb.go
# /proto/course/api/proto/v1/*.pb.go
# /proto/dealer/api/proto/v1/*.pb.go
