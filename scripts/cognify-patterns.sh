#!/usr/bin/env bash
#
# cognify-patterns.sh
#
# Processes datasets through Cognee's cognify API to build knowledge graphs.
# Accepts dataset names via stdin. If no stdin, cognifies all datasets.
#
# Usage:
#   echo "patterns" | ./cognify-patterns.sh         # Single dataset via stdin
#   cat datasets-loaded.txt | ./cognify-patterns.sh # Multiple datasets from file
#   ./cognify-patterns.sh                           # No stdin = cognify all datasets
#
# Environment Variables:
#   COGNEE_URL - Base URL for Cognee API (default: http://localhost:8000)
#
# Output:
#   Logs to ${PROJ_ROOT}/memory-mcp-server/logs/cognify-patterns-${TIMESTAMP}.log

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJ_ROOT="${PROJ_ROOT:-$(cd "${SCRIPT_DIR}/.." && pwd)}"

# Global variable declarations
COGNEE_URL="${COGNEE_URL:-http://localhost:8000}"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
LOG_DIR="${PROJ_ROOT}/memory-mcp-server/logs"
LOG_FILE="${LOG_DIR}/cognify-patterns-${TIMESTAMP}.log"

mkdir -p "${LOG_DIR}"

# Redirect all output to log file
exec > >(tee -a "${LOG_FILE}") 2>&1

printf "Logging to: %s\n" "${LOG_FILE}"

validate_args() {
    if ! command -v curl >/dev/null 2>&1; then
        printf "ERROR: curl is required but not installed\n" >&2
        return 1
    fi

    return 0
}

cognify_dataset() {
    local dataset_name="${1}"
    local cognify_url="${COGNEE_URL}/api/v1/cognify"

    if [ -z "${dataset_name}" ]; then
        printf "ERROR: Dataset name cannot be empty\n" >&2
        return 1
    fi

    printf "Processing dataset '%s' into knowledge graph...\n" "${dataset_name}"

    if ! curl -sf -X POST "${cognify_url}" \
        -H "Content-Type: application/json" \
        -d "{\"datasets\": [\"${dataset_name}\"]}" >/dev/null 2>&1; then
        printf "ERROR: Failed to cognify dataset '%s'\n" "${dataset_name}" >&2
        return 1
    fi

    printf "✓ Knowledge graph processing started for '%s'\n" "${dataset_name}"
    return 0
}

cognify_all() {
    local cognify_url="${COGNEE_URL}/api/v1/cognify"

    printf "No datasets specified. Processing ALL datasets into knowledge graph...\n"

    if ! curl -sf -X POST "${cognify_url}" \
        -H "Content-Type: application/json" \
        -d "{}" >/dev/null 2>&1; then
        printf "ERROR: Failed to cognify all datasets\n" >&2
        return 1
    fi

    printf "✓ Knowledge graph processing started for all datasets\n"
    return 0
}

process_datasets() {
    # Reject command-line arguments
    if [ $# -gt 0 ]; then
        printf "ERROR: This script does not accept command-line arguments\n" >&2
        printf "Usage: cat datasets-file.txt | %s\n" "$(basename "$0")" >&2
        printf "   or: %s  (cognifies all datasets)\n" "$(basename "$0")" >&2
        return 1
    fi

    local dataset_name
    local processed_count=0
    local failed_count=0
    local has_data=0

    # Try to read from stdin
    while IFS= read -r dataset_name; do
        # Skip empty lines
        if [ -z "${dataset_name}" ]; then
            continue
        fi

        has_data=1
        if cognify_dataset "${dataset_name}"; then
            processed_count=$((processed_count + 1))
        else
            failed_count=$((failed_count + 1))
        fi
    done

    # If no data was read from stdin, cognify everything
    if [ "${has_data}" -eq 0 ]; then
        if ! cognify_all; then
            return 1
        fi
    else
        # Show summary for stdin processing
        printf "\n=== Cognify Summary ===\n"
        printf "Successfully started: %d\n" "${processed_count}"
        if [ "${failed_count}" -gt 0 ]; then
            printf "Failed: %d\n" "${failed_count}"
        fi
    fi

    printf "\nNote: Processing runs asynchronously. Check logs with:\n"
    printf "      docker compose -f memory-mcp-server/docker-compose.yaml logs -f cognee-api\n"

    if [ "${failed_count}" -gt 0 ]; then
        return 1
    fi

    return 0
}

main() {
    if ! validate_args; then
        return 1
    fi

    if ! process_datasets "$@"; then
        return 1
    fi

    return 0
}

main "$@"
