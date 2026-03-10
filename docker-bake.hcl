# =============================================================================
# Docker Bake Configuration
# =============================================================================
# Builds all services with shared proto generation.
# Proto files are generated once and shared across all service builds.
#
# Usage:
#   docker buildx bake              # Build all services (parallel)
#   docker buildx bake deck         # Build only deck
#   docker buildx bake --push       # Build and push to registry
#
# Sequential builds for low-end machines:
#   make services                   # Auto-detects resources, uses sequential if needed
#   BUILD_MODE=sequential make services  # Force sequential builds
# =============================================================================

variable "REGISTRY" {
  default = ""
}

variable "TAG" {
  default = "latest"
}

# -----------------------------------------------------------------------------
# Proto Builder - generates all proto files once
# -----------------------------------------------------------------------------
target "proto" {
  dockerfile = "proto.Dockerfile"
  context    = "."
  tags       = ["toshokan-proto:${TAG}"]
}

# -----------------------------------------------------------------------------
# Go Services - all depend on proto target
# -----------------------------------------------------------------------------
target "deck" {
  dockerfile = "deck/Dockerfile"
  context    = "."
  contexts = {
    proto-builder = "target:proto"
  }
  tags = ["${REGISTRY}toshokan-deck:${TAG}"]
}

target "user" {
  dockerfile = "user/Dockerfile"
  context    = "."
  contexts = {
    proto-builder = "target:proto"
  }
  tags = ["${REGISTRY}toshokan-user:${TAG}"]
}

target "course" {
  dockerfile = "course/Dockerfile"
  context    = "."
  contexts = {
    proto-builder = "target:proto"
  }
  tags = ["${REGISTRY}toshokan-course:${TAG}"]
}

target "gate" {
  dockerfile = "gate/Dockerfile"
  context    = "."
  contexts = {
    proto-builder = "target:proto"
  }
  tags = ["${REGISTRY}toshokan-gate:${TAG}"]
}

# -----------------------------------------------------------------------------
# Rust Service - has its own proto generation via tonic-build
# -----------------------------------------------------------------------------
target "dealer" {
  dockerfile = "Dockerfile"
  context    = "dealer/"
  tags       = ["${REGISTRY}toshokan-dealer:${TAG}"]
}

# -----------------------------------------------------------------------------
# Migrations
# -----------------------------------------------------------------------------
target "migrations" {
  dockerfile = "migrations.Dockerfile"
  context    = "."
  tags       = ["${REGISTRY}toshokan-migrations:${TAG}"]
}

# -----------------------------------------------------------------------------
# Groups
# -----------------------------------------------------------------------------
group "default" {
  targets = ["deck", "user", "course", "gate", "dealer"]
}

group "go-services" {
  targets = ["deck", "user", "course", "gate"]
}

group "all" {
  targets = ["deck", "user", "course", "gate", "dealer", "migrations"]
}
