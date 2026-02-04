#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJ_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
SRC_ROOT="$(cd "${PROJ_ROOT}/.." && pwd)"

IMAGE_NAME="${IMAGE_NAME:-ghcr.io/twistingmercury/mnemonic}"
IMAGE_TAG="${IMAGE_TAG:-latest}"

BUILD_VER="${BUILD_VER:-$(git -C "${PROJ_ROOT}" describe --tags --abbrev=0 2>/dev/null || echo 'dev')}"
BUILD_DATE="${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
BUILD_COMMIT="${BUILD_COMMIT:-$(git -C "${PROJ_ROOT}" rev-parse --short HEAD 2>/dev/null || echo 'unknown')}"

LOCAL_BUILD="${LOCAL_BUILD:-0}"

MAX_RETRIES="${MAX_RETRIES:-30}"
RETRY_INTERVAL="${RETRY_INTERVAL:-2}"
PG_HOST="${PG_HOST:-localhost}"
PG_PORT="${PG_PORT:-5433}"
PG_USER="${PG_USER:-mnemonic}"

case "${LOCAL_BUILD}" in
    0|1) ;;
    *) printf "ERROR: LOCAL_BUILD must be 0 or 1, got: %s\n" "${LOCAL_BUILD}" >&2; exit 1 ;;
esac

if [ "${LOCAL_BUILD}" -eq 1 ]; then
    IMAGE_TAG="${BUILD_VER}-localdev"
fi

cleanup_db_infra() {
    printf "Cleaning up database infrastructure...\n" >&2
    docker compose -f "${PROJ_ROOT}/migrations/docker-compose.yaml" down -v --remove-orphans > /dev/null 2>&1 || true
}

start_db_infra() {
    printf "\n=== Starting database infrastructure ===\n"

    if ! docker compose -f "${PROJ_ROOT}/migrations/docker-compose.yaml" up -d; then
        printf "ERROR: Failed to start database infrastructure\n" >&2
        return 1
    fi

    printf "Waiting for PostgreSQL to be ready...\n"
    for i in $(seq 1 "${MAX_RETRIES}"); do
        if pg_isready -h "${PG_HOST}" -p "${PG_PORT}" -U "${PG_USER}" > /dev/null 2>&1; then
            printf "PostgreSQL is ready (attempt %d/%d)\n" "${i}" "${MAX_RETRIES}"
            return 0
        fi

        if [ "${i}" -eq "${MAX_RETRIES}" ]; then
            printf "ERROR: PostgreSQL failed to become ready after %d attempts\n" "${MAX_RETRIES}" >&2
            return 1
        fi

        printf "Waiting for PostgreSQL... (%d/%d)\n" "${i}" "${MAX_RETRIES}"
        sleep "${RETRY_INTERVAL}"
    done

    return 1
}

run_integration_tests() {
    printf "\n=== Running integration tests ===\n"

    printf "Running agent repository integration tests...\n"
    if ! (cd "${PROJ_ROOT}" && go test -tags=integration ./internal/repository/agent/... -v); then
        printf "ERROR: Agent integration tests failed\n" >&2
        return 1
    fi

    printf "\nRunning pattern repository integration tests...\n"
    if ! (cd "${PROJ_ROOT}" && go test -tags=integration ./internal/repository/pattern/... -v); then
        printf "ERROR: Pattern integration tests failed\n" >&2
        return 1
    fi

    printf "\n=== Integration tests passed ===\n"
    return 0
}

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
        docker compose -f "${PROJ_ROOT}/tests/docker-compose.yaml" down -v --remove-orphans > /dev/null 2>&1 || true
    }
    trap cleanup EXIT

    docker compose -f "${PROJ_ROOT}/tests/docker-compose.yaml" up -d --abort-on-container-exit --exit-code-from mnemonic_tests

    trap - EXIT
    cleanup
}

main(){
    trap cleanup_db_infra EXIT

    if ! start_db_infra; then
        printf "ERROR: Failed to start database infrastructure\n" >&2
        exit 1
    fi

    if ! run_integration_tests; then
        printf "ERROR: Integration tests failed, aborting build\n" >&2
        exit 1
    fi

    trap - EXIT
    cleanup_db_infra

    build_api

    if [ "${LOCAL_BUILD}" -eq 1 ]; then
        docker run --rm "${IMAGE_NAME}:${IMAGE_TAG}" --version
    fi

    e2e_tests
}

main "$@"
