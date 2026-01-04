course:
	go build -o bin/course ./cmd/course/ ;

test:
	go test ./... -p=1 -coverprofile=coverage.out *.go

coverage: test
	go tool cover -html=coverage.out

proto:
	protoc --proto_path=api/proto/v1/ --go_out=api/proto/v1/ --go_opt=paths=source_relative api/proto/v1/course.proto
	protoc --go-grpc_opt=require_unimplemented_servers=false --proto_path=api/proto/v1/ --go-grpc_out=api/proto/v1/ --go-grpc_opt=paths=source_relative api/proto/v1/course.proto

migrations:
	go run cmd/migrate/main.go ;
