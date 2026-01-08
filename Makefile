services:
	$(MAKE) -C deck
	$(MAKE) -C user
	$(MAKE) -C gate
	$(MAKE) -C course
	$(MAKE) -C dealer

test-api-schema:
	cd api && ./run_schemathesis.sh && cd -

test-api-basic-flow:
	pytest api

test:
	go test ./... -p=1 -coverprofile=coverage.out *.go

coverage: test
	go tool cover -html=coverage.out

dev: services
	docker compose up --build

proto:
	$(MAKE) proto -C deck
	$(MAKE) proto -C user
	$(MAKE) proto -C course
	$(MAKE) proto -C dealer

migrations:
	$(MAKE) migrations -C deck
	$(MAKE) migrations -C user
	$(MAKE) migrations -C course
