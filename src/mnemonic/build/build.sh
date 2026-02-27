#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJ_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

IMAGE_NAME="${IMAGE_NAME:-ghcr.io/twistingmercury/mnemonic}"
IMAGE_TAG="${IMAGE_TAG:-latest}"

BUILD_VER="${BUILD_VER:-$(git -C "${PROJ_ROOT}" describe --tags --abbrev=0 2>/dev/null || echo 'dev')}"
BUILD_DATE="${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
BUILD_COMMIT="${BUILD_COMMIT:-$(git -C "${PROJ_ROOT}" rev-parse --short HEAD 2>/dev/null || echo 'unknown')}"


build_api(){
    printf "\n=== starting image build ===\n"

    docker build --rm --no-cache \
        --file "${SCRIPT_DIR}/Dockerfile" \
        --build-arg BUILD_VER="${BUILD_VER}" \
        --build-arg BUILD_DATE="${BUILD_DATE}" \
        --build-arg BUILD_COMMIT="${BUILD_COMMIT}" \
        --target final \
        --tag "${IMAGE_NAME}:${IMAGE_TAG}" \
        --tag "${IMAGE_NAME}:latest" \
        "$(cd "${PROJ_ROOT}" && pwd)"

    printf "\nImage: %s:%s\n" "${IMAGE_NAME}" "${IMAGE_TAG}"
    docker images "${IMAGE_NAME}:${IMAGE_TAG}" --format "Size: {{.Size}}"

    return 0
}

e2e_tests(){
    printf "\n=== starting end-to-end tests ===\n"

    local compose_file="${PROJ_ROOT}/tests/docker-compose.yaml"

    cleanup() {
        docker compose -f "${compose_file}" down -v --remove-orphans > /dev/null 2>&1 || true
    }
    trap cleanup EXIT

    printf "Starting infrastructure services...\n"
    if ! docker compose -f "${compose_file}" up -d postgres neo4j; then
        printf "ERROR: Failed to start infrastructure services\n" >&2
        return 1
    fi

    printf "Waiting for infrastructure to be healthy...\n"
    if ! docker compose -f "${compose_file}" run --rm migrate; then
        printf "ERROR: E2E migrations failed\n" >&2
        return 1
    fi

    docker compose -f "${compose_file}" up \
        --build \
        --abort-on-container-exit \
        --exit-code-from mnemonic_tests \
        mnemonic_api mnemonic_tests

    trap - EXIT
    cleanup

    return 0
}

main(){
    build_api

    e2e_tests

    return 0
}

main "$@"
