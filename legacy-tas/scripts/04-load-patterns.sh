#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJ_ROOT="${PROJ_ROOT:-$(cd "${SCRIPT_DIR}/.." && pwd)}"

# shellcheck source=../lib/print.sh
. "${PROJ_ROOT}/lib/print.sh"

# Global variable declarations
API_URL="${API_URL:-http://localhost:8000}"
PATTERNS_DIR="${PATTERNS_DIR:-${PROJ_ROOT}/../agent-patterns}"
TIMESTAMP="${TIMESTAMP:-$(date +%Y%m%d-%H%M%S)}"
LOG_DIR="${SCRIPT_DIR}/logs/${TIMESTAMP}"
LOG_FILE="${LOG_DIR}/04-load-patterns.log"
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

add_pattern() {
    local pattern_file="${1}"
    local dataset_name="${2}"
    local add_url="${API_URL}/api/v1/add"

    if ! curl -sf -X POST "${add_url}" \
        -F "data=@${pattern_file}" \
        -F "datasetName=${dataset_name}" >/dev/null 2>&1; then
        printf "ERROR: Failed to add pattern: %s\n" "${pattern_file}" >&2
        return 1
    fi

    return 0
}

load_dataset() {
    local dataset_name="${1}"
    local dataset_dir="${2}"
    local pattern_count=0
    local pattern_file

    printf "Loading dataset '%s'...\n" "${dataset_name}"

    while IFS= read -r pattern_file; do
        # Skip README and schema reference files
        if [[ "${pattern_file}" == */README.md ]] || [[ "${pattern_file}" == */PATTERN-METADATA-SCHEMA.md ]]; then
            printf "  Skipping: %s\n" "${pattern_file#"${PROJ_ROOT}"/}"
            continue
        fi

        printf "  Adding: %s\n" "${pattern_file#"${PROJ_ROOT}"/}"

        if ! add_pattern "${pattern_file}" "${dataset_name}"; then
            return 1
        fi

        pattern_count=$((pattern_count + 1))
    done < <(find "${dataset_dir}" -type f -name "*.md")

    printf "✓ Added %d patterns to '%s'\n\n" "${pattern_count}" "${dataset_name}"
    return 0
}

load_patterns() {
    local subdir
    local dataset_name
    local total_datasets=0
    local failed_datasets=0

    printf "Loading patterns from %s...\n\n" "${PATTERNS_DIR}"

    # Clear datasets file
    printf "" > "${DATASETS_FILE}"

    # Iterate through top-level subdirectories
    while IFS= read -r subdir; do
        dataset_name="$(basename "${subdir}")"

        if ! load_dataset "${dataset_name}" "${subdir}"; then
            printf "ERROR: Failed to load dataset '%s'\n" "${dataset_name}" >&2
            failed_datasets=$((failed_datasets + 1))
            continue
        fi

        # Write dataset name to file
        printf "%s\n" "${dataset_name}" >> "${DATASETS_FILE}"
        total_datasets=$((total_datasets + 1))
    done < <(find "${PATTERNS_DIR}" -mindepth 1 -maxdepth 1 -type d | sort)

    if [ "${failed_datasets}" -gt 0 ]; then
        printf "WARNING: %d dataset(s) failed to load\n" "${failed_datasets}" >&2
    fi

    printf "✓ Loaded %d dataset(s)\n" "${total_datasets}"
    return 0
}

main() {
    if ! validate_args; then
        return 1
    fi

    if ! load_patterns; then
        return 1
    fi

    return 0
}

main "$@"
