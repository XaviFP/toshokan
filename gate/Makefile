gate:
	go build -o bin/gate ./cmd/gate/

test:
	go test ./... -p=1 -coverprofile=coverage.out *.go

coverage: test
	go tool cover -html=coverage.out
