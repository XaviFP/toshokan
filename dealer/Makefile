dealer:
	cargo build ;

proto:
	protoc --proto_path=api/proto/v1/ --go_out=api/proto/v1/ --go_opt=paths=source_relative dealer.proto
	protoc --go-grpc_opt=require_unimplemented_servers=false  --proto_path=api/proto/v1/ --go-grpc_out=api/proto/v1/ --go-grpc_opt=paths=source_relative dealer.proto

