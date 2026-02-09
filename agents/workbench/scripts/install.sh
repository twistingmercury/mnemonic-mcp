#!/usr/bin/env bash

set -e

# shellcheck disable=SC1091

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJ_ROOT="${PROJ_ROOT:-$(cd "${SCRIPT_DIR}/.." && pwd)}"

# Set shared timestamp for all logs during this install run
export TIMESTAMP="${TIMESTAMP:-$(date +%Y%m%d-%H%M%S)}"

# shellcheck source=../lib/print.sh
. "${PROJ_ROOT}/lib/print.sh"

main(){
    print::info "Starting agent workbench installation..."

    print::info "Step 1/7: Starting memory infrastructure..."
    if ! "${SCRIPT_DIR}/00-start-memory-infra.sh"; then
        print::error "Failed to start memory infrastructure"
        return 1
    fi

    print::info "Step 2/7: Installing agent definitions..."
    if ! "${SCRIPT_DIR}/01-install-agents.sh"; then
        print::error "Failed to install agent definitions"
        return 2
    fi

    print::info "Step 3/7: Installing skills..."
    if ! "${SCRIPT_DIR}/02-install-skills.sh"; then
        print::error "Failed to install skills"
        return 3
    fi

    print::info "Step 4/7: Installing global agent rules..."
    if ! "${SCRIPT_DIR}/03-install-global-agent-rules.sh"; then
        print::error "Failed to install global agent rules"
        return 4
    fi

    print::info "Step 5/7: Validating pattern metadata..."
    if ! "${SCRIPT_DIR}/04-validate-metadata.sh"; then
        print::error "Failed to validate metadata"
        return 5
    fi

    print::info "Step 6/7: Loading patterns..."
    if ! "${SCRIPT_DIR}/05-load-patterns.sh"; then
        print::error "Failed to load patterns"
        return 6
    fi

    print::info "Step 7/7: Enriching patterns with relationships..."

    if [ ! -f "${SCRIPT_DIR}/logs/${TIMESTAMP}/datasets-loaded.txt" ]; then
        print::error "expected file datasets-loaded.txt not found"
        return 7
    fi

    mapfile -t datasets < "${SCRIPT_DIR}/logs/${TIMESTAMP}/datasets-loaded.txt"

    failed_count=0
    for ds in "${datasets[@]}"; do
        if ! echo "${ds}" | "${SCRIPT_DIR}/06-enrich-patterns.sh"; then
            print::error "Failed to enrich dataset ${ds}"
            failed_count=$((failed_count + 1))
        else
            print::success "Successfully enriched dataset ${ds}"
        fi
    done

    if [ "${failed_count}" -gt 0 ]; then
        print::error "Failed to enrich ${failed_count} dataset(s)"
        return 8
    fi

    return 0
}

main "$@"