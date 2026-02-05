#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJ_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COGNEE_DIR="${PROJ_ROOT}"
TIMESTAMP="${TIMESTAMP:-$(date +%Y%m%d-%H%M%S)}"
LOG_DIR="${SCRIPT_DIR}/logs/${TIMESTAMP}"
LOG_FILE="${LOG_DIR}/00-start-memory-infra.log"

mkdir -p "${LOG_DIR}"

cd "${COGNEE_DIR}"

printf "Logging to: %s\n" "${LOG_FILE}"

if [ -z "${OPENAI_COGNEE_API_KEY}" ]; then
    printf "ERROR: OPENAI_COGNEE_API_KEY is not set\n" >&2
    exit 1
fi
export OPENAI_COGNEE_API_KEY

docker compose -f docker-compose.yaml up -d >> "${LOG_FILE}" 2>&1

printf "Waiting for Cognee servers to be healthy...\n"
max_attempts=30
attempt=0
while [ $attempt -lt $max_attempts ]; do
    if docker compose -f docker-compose.yaml ps cognee-mcp | grep -q "healthy" && \
       docker compose -f docker-compose.yaml ps cognee-api | grep -q "healthy"; then
        break
    fi
    attempt=$((attempt + 1))
    if [ $attempt -eq $max_attempts ]; then
        printf "ERROR: Cognee servers failed to start\n" >&2
        printf "Check logs: %s\n" "${LOG_FILE}" >&2
        exit 1
    fi
    sleep 5
done

printf "Checking Cognee health endpoints...\n"
if curl -sf --max-time 10 http://localhost:4000/health > /dev/null && \
   curl -sf --max-time 10 http://localhost:8000/health > /dev/null; then
    printf "✓ Cognee server is healthy (MCP: port 4000, REST API: port 8000)\n"
else
    printf "ERROR: Cognee health check failed\n" >&2
    printf "Check logs: %s\n" "${LOG_FILE}" >&2
    exit 1
fi

printf "\n✓ Setup completed successfully\n"
