services:
	$(MAKE) -C deck
	$(MAKE) -C user
	$(MAKE) -C gate
	$(MAKE) -C dealer

test:
	go test ./... -p=1 -coverprofile=coverage.out *.go

coverage: test
	go tool cover -html=coverage.out

dev: services
	docker compose up --build

proto:
	$(MAKE) proto -C deck
	$(MAKE) proto -C user
	$(MAKE) proto -C dealer

migrations:
	$(MAKE) migrations -C deck
	$(MAKE) migrations -C user