#!/usr/bin/env bash
#
# load-patterns.sh
#
# Loads pattern files into Cognee via the /api/v1/add endpoint.
# Validates patterns before loading and outputs the dataset name for piping.
#
# Usage:
#   ./load-patterns.sh                              # Load patterns (writes dataset name to file)
#   cat memory-mcp-server/logs/datasets-loaded.txt | ./cognify-patterns.sh  # Cognify loaded datasets
#
# Environment Variables:
#   COGNEE_URL    - Base URL for Cognee API (default: http://localhost:8000)
#   PATTERNS_DIR  - Directory containing pattern files (default: ${PROJ_ROOT}/agent-patterns)
#   DATASET_NAME  - Name of the dataset to load into (default: patterns)
#
# Output:
#   Logs to ${PROJ_ROOT}/memory-mcp-server/logs/load-patterns-${TIMESTAMP}.log
#   Writes dataset name to ${PROJ_ROOT}/memory-mcp-server/logs/datasets-loaded.txt

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJ_ROOT="${PROJ_ROOT:-$(cd "${SCRIPT_DIR}/.." && pwd)}"

# Global variable declarations
COGNEE_URL="${COGNEE_URL:-http://localhost:8000}"
PATTERNS_DIR="${PATTERNS_DIR:-${PROJ_ROOT}/agent-patterns}"
DATASET_NAME="${DATASET_NAME:-patterns}"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
LOG_DIR="${PROJ_ROOT}/memory-mcp-server/logs"
LOG_FILE="${LOG_DIR}/load-patterns-${TIMESTAMP}.log"
DATASETS_FILE="${LOG_DIR}/datasets-loaded.txt"

mkdir -p "${LOG_DIR}"

# Redirect all output to log file
exec > >(tee -a "${LOG_FILE}") 2>&1

printf "Logging to: %s\n" "${LOG_FILE}"

validate_args() {
    if [ ! -d "${PATTERNS_DIR}" ]; then
        printf "ERROR: Patterns directory not found: %s\n" "${PATTERNS_DIR}" >&2
        return 1
    fi

    if ! command -v curl >/dev/null 2>&1; then
        printf "ERROR: curl is required but not installed\n" >&2
        return 1
    fi

    return 0
}

check_cognee_health() {
    local health_url="${COGNEE_URL}/health"

    if ! curl -sf --max-time 30 "${health_url}" >/dev/null 2>&1; then
        printf "ERROR: Cognee server not reachable at %s\n" "${COGNEE_URL}" >&2
        printf "       Start with: cd memory-mcp-server && docker compose up -d\n" >&2
        return 1
    fi

    printf "✓ Cognee server is reachable\n"
    return 0
}

validate_patterns() {
    local validation_script="${SCRIPT_DIR}/validate-metadata.sh"

    if [ ! -f "${validation_script}" ]; then
        printf "ERROR: Validation script not found: %s\n" "${validation_script}" >&2
        return 1
    fi

    printf "Validating pattern metadata...\n"
    if ! bash "${validation_script}"; then
        printf "ERROR: Pattern validation failed\n" >&2
        return 1
    fi

    printf "✓ Pattern validation passed\n"
    return 0
}

add_pattern_to_cognee() {
    local pattern_file="${1}"
    local add_url="${COGNEE_URL}/api/v1/add"

    if ! curl -sf -X POST "${add_url}" \
        -F "data=@${pattern_file}" \
        -F "datasetName=${DATASET_NAME}" >/dev/null 2>&1; then
        printf "ERROR: Failed to add pattern: %s\n" "${pattern_file}" >&2
        return 1
    fi

    return 0
}

load_patterns() {
    local pattern_count=0
    local pattern_file

    printf "Loading patterns from %s...\n" "${PATTERNS_DIR}"

    while IFS= read -r pattern_file; do
        # Skip README and schema reference files
        if [[ "${pattern_file}" == */README.md ]] || [[ "${pattern_file}" == */PATTERN-METADATA-SCHEMA.md ]]; then
            printf "  Skipping: %s\n" "${pattern_file#"${PROJ_ROOT}"/}"
            continue
        fi

        printf "  Adding: %s\n" "${pattern_file#"${PROJ_ROOT}"/}"

        if ! add_pattern_to_cognee "${pattern_file}"; then
            return 1
        fi

        pattern_count=$((pattern_count + 1))
    done < <(find "${PATTERNS_DIR}" -type f -name "*.md")

    printf "✓ Added %d patterns\n" "${pattern_count}"
    return 0
}

main() {
    if ! validate_args; then
        return 1
    fi

    if ! validate_patterns; then
        return 1
    fi

    if ! load_patterns; then
        return 1
    fi

    # Write dataset name to file for cognify script
    printf "%s\n" "${DATASET_NAME}" > "${DATASETS_FILE}"

    printf "\n=== Pattern Loading Complete ===\n"
    printf "Patterns have been added to dataset '%s'.\n" "${DATASET_NAME}"
    printf "\nDataset names written to: %s\n" "${DATASETS_FILE}"
    printf "\nTo process into knowledge graph, run:\n"
    printf "  cat %s | ./cognify-patterns.sh\n" "${DATASETS_FILE}"

    return 0
}

main "$@"
