export PATH=$(shell printenv PATH):$(shell go env GOPATH)/bin

deck:
	go build -o bin/deck ./cmd/deck/

test:
	go test ./... -p=1 -coverprofile=coverage.out *.go

coverage: test
	go tool cover -html=coverage.out

proto:
	protoc --proto_path=api/proto/v1/ --go_out=api/proto/v1/ --go_opt=paths=source_relative deck.proto
	protoc --go-grpc_opt=require_unimplemented_servers=false  --proto_path=api/proto/v1/ --go-grpc_out=api/proto/v1/ --go-grpc_opt=paths=source_relative deck.proto

migrations:
	go run cmd/migrate/main.go
