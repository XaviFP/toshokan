user:
	go build -o bin/user ./cmd/user/ ;

test:
	go test ./... -p=1 -coverprofile=coverage.out *.go

coverage: test
	go tool cover -html=coverage.out

proto:
	protoc --proto_path=api/proto/v1/ --go_out=api/proto/v1/ --go_opt=paths=source_relative user.proto
	protoc --go-grpc_opt=require_unimplemented_servers=false  --proto_path=api/proto/v1/ --go-grpc_out=api/proto/v1/ --go-grpc_opt=paths=source_relative user.proto

migrations:
	go run cmd/migrate/main.go ;
