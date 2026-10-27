# =============================================================================
# Docker-only Build System
# =============================================================================
# No local dependencies required (go, rust, protoc, etc.)
# Everything is built inside Docker containers
# =============================================================================

.PHONY: clean rebuild test-api-schema test-api-basic-flow test test-dealer sqlx-prepare coverage dev proto migrations

# Clean up containers, images, and build artifacts
clean:
	docker compose down --rmi local -v
	rm -f user/bin/* gate/bin/* deck/bin/* course/bin/* dealer/target/release/dealer

# Rebuild: stop, remove, and rebuild application services
rebuild: clean
	docker compose stop user gate deck dealer course
	docker compose rm -f user gate deck dealer course
	docker compose up --build -d user gate deck dealer course

# Run API schema tests (schemathesis)
test-api-schema:
	cd api && ./run_schemathesis.sh && cd -

# Run API basic flow tests
test-api-basic-flow:
	pytest api

# Run unit and integration tests in Docker
test:
	docker run --rm -v $(PWD):/app -w /app --network host \
		golang:1.24-bookworm \
		go test ./... -p=1 -coverprofile=coverage.out

# dealer's tests and query cache run in the `dev` stage of dealer/Dockerfile (Rust, protoc, sqlx-cli),
# on the host network, against the db container. Credentials come from env/.common.
DB_USER := $(shell sed -n 's/^DB_USER=//p' env/.common)
DB_PASSWORD := $(shell sed -n 's/^DB_PASSWORD=//p' env/.common)
DB_PORT := $(shell sed -n 's/^DB_PORT=//p' env/.common)
DEALER_DATABASE_URL = postgres://$(DB_USER):$(DB_PASSWORD)@localhost:$(DB_PORT)/deck
DEALER_DEV = docker build -q -f dealer/Dockerfile --target dev -t toshokan-dealer-dev . > /dev/null && \
	docker run --rm -v $(PWD):/app --network host \
		-v toshokan-dealer-cargo-registry:/usr/local/cargo/registry \
		-v toshokan-dealer-target:/target -e CARGO_TARGET_DIR=/target \
		-e DATABASE_URL=$(DEALER_DATABASE_URL) toshokan-dealer-dev

# Run dealer's unit and integration tests; needs the db container running and `make migrations`.
# Fails first if the committed query cache (dealer/.sqlx) is out of date with the deck schema.
# Each DB test gets its own temporary database, created through DATABASE_URL's user.
test-dealer:
	$(DEALER_DEV) sh -c "cargo sqlx prepare --check -- --all-targets && SQLX_OFFLINE=true cargo test"

# Regenerate dealer's query cache (dealer/.sqlx) after changing a query; needs `make migrations`
sqlx-prepare:
	$(DEALER_DEV) sh -c "cargo sqlx prepare -- --all-targets && chown -R $(shell id -u):$(shell id -g) .sqlx"

# Generate coverage report (requires test to have run first)
coverage: test
	docker run --rm -v $(PWD):/app -w /app \
		golang:1.24-bookworm \
		go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Start development environment
dev:
	docker compose up --build

# Stop all containers
stop:
	docker compose down

# Regenerate the committed .pb.go files after changing a .proto
proto:
	@docker build -q --target protoc -t toshokan-protoc . > /dev/null
	docker run --rm -v $(PWD):/app -w /app --user $(shell id -u):$(shell id -g) \
		toshokan-protoc sh scripts/generate-protos.sh

# =============================================================================
# Migrations
# =============================================================================

# Run all database migrations
migrations:
	@docker build -f migrations.Dockerfile -t toshokan-migrations . > /dev/null
	@docker run --rm --network toshokan \
		--env-file env/.common \
		-e SERVICE=all \
		toshokan-migrations

# Run migrations for a specific service (e.g., make migrate-deck)
migrate-%:
	@docker build -f migrations.Dockerfile -t toshokan-migrations . > /dev/null
	@docker run --rm --network toshokan \
		--env-file env/.common \
		-e SERVICE=$* \
		-e DB_NAME=$$(if [ "$*" = "user" ]; then echo "users"; else echo "$*"; fi) \
		toshokan-migrations
