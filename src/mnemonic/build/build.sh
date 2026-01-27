#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJ_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

IMAGE_NAME="${IMAGE_NAME:-ghcr.io/twistingmercury/mnemonic}"
IMAGE_TAG="${IMAGE_TAG:-latest}"

BUILD_VER="${BUILD_VER:-$(git -C "${PROJ_ROOT}" describe --tags --abbrev=0 2>/dev/null || echo 'dev')}"
BUILD_DATE="${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
BUILD_COMMIT="${BUILD_COMMIT:-$(git -C "${PROJ_ROOT}" rev-parse --short HEAD 2>/dev/null || echo 'unknown')}"

LOCAL_BUILD="${LOCAL_BUILD:-0}"

if [ "${LOCAL_BUILD}" -eq 1 ]; then
    IMAGE_TAG="${BUILD_VER}-localdev"
fi

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
}

e2e_tests(){
    printf "\n=== starting end-to-end tests ===\n"

    cleanup() {
        docker compose -f "${PROJ_ROOT}/tests/docker-compose.yaml" down --remove-orphans > /dev/null 2>&1 || true
    }
    trap cleanup EXIT

    docker compose -f "${PROJ_ROOT}/tests/docker-compose.yaml" up --exit-code-from mnemonic_tests

    trap - EXIT
    cleanup
}

main(){
    build_api

    if [ "${LOCAL_BUILD}" -eq 1 ]; then
        docker run --rm "${IMAGE_NAME}:${IMAGE_TAG}" --version
    fi

    e2e_tests
}

main "$@"
