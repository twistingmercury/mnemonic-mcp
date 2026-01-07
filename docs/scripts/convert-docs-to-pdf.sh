#!/usr/bin/env bash
#
# convert-docs-to-pdf.sh
#
# Converts all markdown files in the architecture documentation directory to PDF.
# Uses pandoc with mermaid-filter for diagram rendering and XeLaTeX for font support.
#
# Usage:
#   ./convert-docs-to-pdf.sh
#
# Environment Variables:
#   DOCS_DIR   - Directory containing markdown files (default: docs/architecture)
#   OUTPUT_DIR - Directory for PDF output (default: docs/publish)
#
# Output:
#   PDF files in docs/publish directory
#

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJ_ROOT="${PROJ_ROOT:-$(cd "${SCRIPT_DIR}/.." && pwd)}"

# Global variable declarations
DOCS_DIR="${DOCS_DIR:-${PROJ_ROOT}/architecture}"
OUTPUT_DIR="${OUTPUT_DIR:-${PROJ_ROOT}/publish}"

validate_args() {
    if ! command -v pandoc >/dev/null 2>&1; then
        printf "ERROR: pandoc is required but not installed\n" >&2
        printf "Install with: brew install pandoc\n" >&2
        return 1
    fi

    if ! command -v xelatex >/dev/null 2>&1; then
        printf "ERROR: xelatex is required but not installed\n" >&2
        printf "Install with: brew install --cask mactex\n" >&2
        return 1
    fi

    if [ ! -d "${DOCS_DIR}" ]; then
        printf "ERROR: Documentation directory does not exist: %s\n" "${DOCS_DIR}" >&2
        return 1
    fi

    # Create output directory if it doesn't exist
    if [ ! -d "${OUTPUT_DIR}" ]; then
        printf "Creating output directory: %s\n" "${OUTPUT_DIR}"
        mkdir -p "${OUTPUT_DIR}"
    fi

    return 0
}

convert_file() {
    local input_file="${1}"
    local filename
    local output_file

    filename="$(basename "${input_file}")"
    output_file="${OUTPUT_DIR}/${filename%.md}.pdf"

    printf "Converting: %s\n" "${filename}"

    if ! pandoc "${input_file}" \
        -F mermaid-filter \
        --pdf-engine=xelatex \
        -V mainfont="Helvetica" \
        -V geometry:margin=25mm \
        -o "${output_file}" 2>&1; then
        printf "  ERROR: Failed to convert %s\n" "${filename}" >&2
        return 1
    fi

    printf "  Created: %s\n" "$(basename "${output_file}")"
    return 0
}

process_files() {
    local file
    local converted_count=0
    local failed_count=0
    local total_count=0

    # Count total files first
    for file in "${DOCS_DIR}"/*.md; do
        if [ -f "${file}" ]; then
            total_count=$((total_count + 1))
        fi
    done

    if [ "${total_count}" -eq 0 ]; then
        printf "No markdown files found in: %s\n" "${DOCS_DIR}"
        return 0
    fi

    printf "Found %d markdown file(s) in: %s\n\n" "${total_count}" "${DOCS_DIR}"

    for file in "${DOCS_DIR}"/*.md; do
        if [ ! -f "${file}" ]; then
            continue
        fi

        if convert_file "${file}"; then
            converted_count=$((converted_count + 1))
        else
            failed_count=$((failed_count + 1))
        fi
    done

    printf "\n=== Conversion Summary ===\n"
    printf "Successfully converted: %d\n" "${converted_count}"

    if [ "${failed_count}" -gt 0 ]; then
        printf "Failed: %d\n" "${failed_count}"
    fi

    if [ "${failed_count}" -gt 0 ]; then
        return 1
    fi

    return 0
}

main() {
    if ! validate_args; then
        return 1
    fi

    if ! process_files; then
        return 1
    fi

    return 0
}

main "$@"
