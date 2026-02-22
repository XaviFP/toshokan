# =============================================================================
# Docker-only Build System
# =============================================================================
# No local dependencies required (go, rust, protoc, etc.)
# Everything is built inside Docker containers
#
# Resource Control:
# Build automatically detects system resources and adjusts strategy.
# Low-end machines (< 3GB RAM or <= 4 cores) build services sequentially.
# Override: BUILD_MODE=sequential make services
# =============================================================================

.PHONY: services clean rebuild test-api-schema test-api-basic-flow test coverage dev proto migrations

# Build all services (uses docker buildx bake for shared proto generation)
# Auto-detects system resources and adjusts parallelism for low-end machines
services:
	@eval $$(./scripts/detect-resources.sh) && \
	if [ "$$BUILD_MODE" = "sequential" ]; then \
		echo "Building services sequentially (low-resource mode)..." && \
		for target in proto deck user course gate dealer; do \
			echo "==> Building $$target..." && \
			docker buildx bake $$target || exit 1; \
		done; \
	else \
		echo "Building services in parallel..." && \
		docker buildx bake; \
	fi

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

# Generate coverage report (requires test to have run first)
coverage: test
	docker run --rm -v $(PWD):/app -w /app \
		golang:1.24-bookworm \
		go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Start development environment
dev:
	docker compose up --build

# Start production environment
prod:
	docker compose --env-file .env.prod -f docker-compose.prod.yaml up --build -d

# Stop all containers
stop:
	docker compose down

# Generate proto files (builds proto image, useful for IDE support)
proto:
	docker build -f proto.Dockerfile -t toshokan-proto .
	@echo "Proto files generated in toshokan-proto image"

# =============================================================================
# Migrations
# =============================================================================

# Run all database migrations
migrations:
	@docker build -f migrations.Dockerfile -t toshokan-migrations . > /dev/null
	@docker run --rm --network toshokan \
		-e DB_HOST=db \
		-e DB_PORT=5432 \
		-e DB_USER=toshokan \
		-e DB_PASSWORD=t.o.s.h.o.k.a.n. \
		-e SERVICE=all \
		toshokan-migrations

# Run migrations for a specific service (e.g., make migrate-deck)
migrate-%:
	@docker build -f migrations.Dockerfile -t toshokan-migrations . > /dev/null
	@docker run --rm --network toshokan \
		-e DB_HOST=db \
		-e DB_PORT=5432 \
		-e DB_USER=toshokan \
		-e DB_PASSWORD=t.o.s.h.o.k.a.n. \
		-e SERVICE=$* \
		-e DB_NAME=$$(if [ "$*" = "user" ]; then echo "users"; else echo "$*"; fi) \
		toshokan-migrations
