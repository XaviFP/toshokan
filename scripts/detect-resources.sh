#!/bin/bash
# =============================================================================
# Resource Detection Script
# =============================================================================
# Detects system resources and outputs optimal build settings.
# Used by Makefile to auto-adjust parallelism for low-end machines.
#
# Usage:
#   eval $(./scripts/detect-resources.sh)
#   # Now BAKE_PARALLEL, GOMAXPROCS, CARGO_BUILD_JOBS are set
# =============================================================================

# Get number of CPU cores
CPU_CORES=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)

# Get available RAM in MB
if [[ -f /proc/meminfo ]]; then
    RAM_MB=$(awk '/MemAvailable/ {print int($2/1024)}' /proc/meminfo)
    # Fallback to MemTotal if MemAvailable not present
    [[ -z "$RAM_MB" || "$RAM_MB" -eq 0 ]] && RAM_MB=$(awk '/MemTotal/ {print int($2/1024)}' /proc/meminfo)
elif command -v sysctl &>/dev/null; then
    # macOS
    RAM_MB=$(($(sysctl -n hw.memsize 2>/dev/null || echo 0) / 1024 / 1024))
else
    RAM_MB=8192  # Default to 8GB if detection fails
fi

# Thresholds
LOW_RAM_THRESHOLD=3072    # 3GB - below this is "low memory"
LOW_CORES_THRESHOLD=4     # 4 cores or fewer is "low CPU"

# Determine if this is a low-resource machine
IS_LOW_RESOURCE=false
if [[ $RAM_MB -lt $LOW_RAM_THRESHOLD ]] || [[ $CPU_CORES -le $LOW_CORES_THRESHOLD ]]; then
    IS_LOW_RESOURCE=true
fi

# Calculate optimal settings
if [[ "$IS_LOW_RESOURCE" == "true" ]]; then
    # Low-resource machine: build services sequentially
    # Each build uses all available cores (Docker default)
    BUILD_MODE=sequential
else
    # High-resource machine: build all services in parallel
    BUILD_MODE=parallel
fi

# Output as shell variable assignments (for eval)
echo "export BUILD_MODE=$BUILD_MODE"
echo "export DETECTED_RAM_MB=$RAM_MB"
echo "export DETECTED_CPU_CORES=$CPU_CORES"
echo "export IS_LOW_RESOURCE=$IS_LOW_RESOURCE"

# Also print info to stderr for visibility
>&2 echo "=== Build Resource Detection ==="
>&2 echo "CPU Cores: $CPU_CORES"
>&2 echo "Available RAM: ${RAM_MB}MB"
>&2 echo "Low-resource mode: $IS_LOW_RESOURCE"
>&2 echo "Build mode: $BUILD_MODE"
>&2 echo "================================"
