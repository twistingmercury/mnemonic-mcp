#!/usr/bin/env bash
#
# 05-enrich-patterns.sh
#
# Processes datasets through the enrichment API to build knowledge graphs.
# Accepts dataset names via stdin. If no stdin, enriches all datasets.
#
# Usage:
#   echo "patterns" | ./05-enrich-patterns.sh         # Single dataset via stdin
#   cat datasets-loaded-TIMESTAMP.txt | ./05-enrich-patterns.sh # Multiple datasets from file
#   ./05-enrich-patterns.sh                           # No stdin = enrich all datasets
#
# Environment Variables:
#   API_URL - Base URL for the API (default: http://localhost:8000)
#
# Output:
#   Logs to ${SCRIPT_DIR}/logs/${TIMESTAMP}/05-enrich-patterns.log

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJ_ROOT="${PROJ_ROOT:-$(cd "${SCRIPT_DIR}/.." && pwd)}"

# Global variable declarations
API_URL="${API_URL:-http://localhost:8000}"
TIMESTAMP="${TIMESTAMP:-$(date +%Y%m%d-%H%M%S)}"
LOG_DIR="${SCRIPT_DIR}/logs/${TIMESTAMP}"
LOG_FILE="${LOG_DIR}/05-enrich-patterns.log"

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

enrich_dataset() {
    local dataset_name="${1}"
    local enrich_url="${API_URL}/api/v1/cognify"

    if [ -z "${dataset_name}" ]; then
        printf "ERROR: Dataset name cannot be empty\n" >&2
        return 1
    fi

    printf "Processing dataset '%s' into knowledge graph...\n" "${dataset_name}"

    if ! curl -sf -X POST "${enrich_url}" \
        -H "Content-Type: application/json" \
        -d "{\"datasets\": [\"${dataset_name}\"]}" >/dev/null 2>&1; then
        printf "ERROR: Failed to enrich dataset '%s'\n" "${dataset_name}" >&2
        return 1
    fi

    printf "✓ Knowledge graph processing started for '%s'\n" "${dataset_name}"
    return 0
}

enrich_all() {
    local enrich_url="${API_URL}/api/v1/cognify"

    printf "No datasets specified. Processing ALL datasets into knowledge graph...\n"

    if ! curl -sf -X POST "${enrich_url}" \
        -H "Content-Type: application/json" \
        -d "{}" >/dev/null 2>&1; then
        printf "ERROR: Failed to enrich all datasets\n" >&2
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
        printf "   or: %s  (enriches all datasets)\n" "$(basename "$0")" >&2
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
        if enrich_dataset "${dataset_name}"; then
            processed_count=$((processed_count + 1))
        else
            failed_count=$((failed_count + 1))
        fi
    done

    # If no data was read from stdin, enrich everything
    if [ "${has_data}" -eq 0 ]; then
        if ! enrich_all; then
            return 1
        fi
    else
        # Show summary for stdin processing
        printf "\n=== Enrich Summary ===\n"
        printf "Successfully started: %d\n" "${processed_count}"
        if [ "${failed_count}" -gt 0 ]; then
            printf "Failed: %d\n" "${failed_count}"
        fi
    fi

    printf "\nNote: Processing runs asynchronously. Check logs with:\n"
    printf "      docker compose logs -f cognee-api\n"

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
