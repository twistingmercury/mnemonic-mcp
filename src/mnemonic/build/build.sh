#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJ_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

IMAGE_NAME="${IMAGE_NAME:-ghcr.io/twistingmercury/mnemonic}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
BRANCH_NAME="${BRANCH_NAME:-}"

BUILD_VER="${BUILD_VER:-$(git -C "${PROJ_ROOT}" describe --tags --abbrev=0 2>/dev/null || echo 'dev')}"
BUILD_DATE="${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
BUILD_COMMIT="${BUILD_COMMIT:-$(git -C "${PROJ_ROOT}" rev-parse --short HEAD 2>/dev/null || echo 'unknown')}"

LOCAL_BUILD="${LOCAL_BUILD:-0}"

if [ "${LOCAL_BUILD}" -eq 1 ]; then
    IMAGE_TAG="${BUILD_VER}-localdev"
fi

build_api(){
    # linting, scans, and unit tests will be ran inside this container build
    printf "\n=== starting image build ===\n"

    local tags=("--tag" "${IMAGE_NAME}:${IMAGE_TAG}")
    if [ "${BRANCH_NAME}" = "main" ]; then
        tags+=("--tag" "${IMAGE_NAME}:latest")
    fi

    docker build --rm --no-cache \
        --file "${SCRIPT_DIR}/Dockerfile" \
        --build-arg BUILD_VER="${BUILD_VER}" \
        --build-arg BUILD_DATE="${BUILD_DATE}" \
        --build-arg BUILD_COMMIT="${BUILD_COMMIT}" \
        --target final \
        "${tags[@]}" \
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

    trap - EXIT  # Clear the trap
    cleanup      # Run cleanup explicitly on success
}

push_image(){
    printf "\n=== pushing image to registry ===\n"
    docker push "${IMAGE_NAME}:${IMAGE_TAG}"

    if [ "${BRANCH_NAME}" = "main" ]; then
        docker push "${IMAGE_NAME}:latest"
        printf "Pushed: %s:%s and %s:latest\n" "${IMAGE_NAME}" "${IMAGE_TAG}" "${IMAGE_NAME}"
    else
        printf "Pushed: %s:%s (latest tag only pushed from main branch)\n" "${IMAGE_NAME}" "${IMAGE_TAG}"
    fi
}

main(){
    build_api

    if [ "${LOCAL_BUILD}" -eq 1 ]; then
        docker run --rm "${IMAGE_NAME}:${IMAGE_TAG}" --version
    fi

    e2e_tests

    if [ "${LOCAL_BUILD}" -ne 1 ]; then
        push_image
    fi
}

main "$@"