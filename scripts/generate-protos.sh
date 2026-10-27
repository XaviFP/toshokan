#!/bin/sh
# Regenerates the committed .pb.go files. Runs in the Dockerfile's protoc stage, through `make proto`.
set -e

for service in deck user course dealer; do
    protoc --proto_path=$service/api/proto/v1/ \
        --proto_path=/usr/include \
        --go_out=$service/api/proto/v1/ --go_opt=paths=source_relative \
        --go-grpc_out=$service/api/proto/v1/ --go-grpc_opt=paths=source_relative \
        --go-grpc_opt=require_unimplemented_servers=false \
        $service.proto
done
