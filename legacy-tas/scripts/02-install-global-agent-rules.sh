#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJ_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Logging setup
TIMESTAMP="${TIMESTAMP:-$(date +%Y%m%d-%H%M%S)}"
LOG_DIR="${SCRIPT_DIR}/logs/${TIMESTAMP}"
LOG_FILE="${LOG_DIR}/02-install-global-agent-rules.log"

mkdir -p "${LOG_DIR}"
exec > >(tee -a "${LOG_FILE}") 2>&1

printf "Logging to: %s\n" "${LOG_FILE}"

CLAUDE_ROOT="${CLAUDE_ROOT:-${HOME}/.claude}"
GLOBAL_CONF="${CLAUDE_ROOT}/CLAUDE.md"
AGENT_RULES_SOURCE="${PROJ_ROOT}/../agents/global-agent-rules.txt"

source "${SCRIPT_DIR}/../lib/print.sh"

## Not every Claude Code install may have a global Claude.md file.
## So when that situation is encountered, we'll need to create it for the user.
create_global_claude_config(){
    print::info "no global CLAUDE.md file exists - creating"

    touch "${GLOBAL_CONF}" || {
        print::error "failed to create the new global config: ${GLOBAL_CONF}"
        return 1
    }

    printf "# CLAUDE.md\n" > "${GLOBAL_CONF}" || {
        print::error "failed to write to new global config: ${GLOBAL_CONF}"
        return 1
    }

    print::success "global CLAUDE.md file created: ${GLOBAL_CONF}"

    return 0
}

## If the user has global Claude.md, we need to back it up in case
## something goes wrong, so it can be restored, even if manually.
backup_global_claude_config(){
    local backup="${GLOBAL_CONF}.backup"

    cp "${GLOBAL_CONF}" "${backup}" || {
        print::error "failed to backup the global config: ${GLOBAL_CONF}"
        return 1
    }

    return 0
}

## Extract the date from agent rules content
## Returns the date string in YYYY-MM-DD format or empty if not found
extract_rules_date() {
    local file="${1}"

    if [ ! -f "${file}" ]; then
        return 0
    fi

    # Look for "**Last Updated: YYYY-MM-DD**" pattern
    grep -E '^\*\*Last Updated: [0-9]{4}-[0-9]{2}-[0-9]{2}\*\*$' "${file}" | \
        sed -E 's/^\*\*Last Updated: ([0-9]{4}-[0-9]{2}-[0-9]{2})\*\*$/\1/' | \
        head -n 1

    return 0
}

## Check if agent rules section exists in the file
has_agent_rules() {
    local file="${1}"

    if [ ! -f "${file}" ]; then
        return 1
    fi

    grep -q "<!-- BEGIN AGENT RULES -->" "${file}"
    return $?
}

## Remove existing agent rules section from CLAUDE.md
## Uses markers: <!-- BEGIN AGENT RULES --> to <!-- END AGENT RULES -->
remove_existing_agent_rules() {
    local claude_file="${1}"

    if [ ! -f "${claude_file}" ]; then
        print::error "file does not exist: ${claude_file}"
        return 1
    fi

    # Create temporary file
    local temp_file
    temp_file="$(mktemp)" || {
        print::error "failed to create temporary file"
        return 1
    }

    # Remove content between markers (inclusive)
    # This uses sed to delete from BEGIN to END markers
    sed '/<!-- BEGIN AGENT RULES -->/,/<!-- END AGENT RULES -->/d' "${claude_file}" > "${temp_file}" || {
        rm -f "${temp_file}"
        print::error "failed to remove agent rules section"
        return 1
    }

    # Replace original file with modified content
    mv "${temp_file}" "${claude_file}" || {
        rm -f "${temp_file}"
        print::error "failed to update ${claude_file}"
        return 1
    }

    return 0
}

## Compare dates in YYYY-MM-DD format
## Returns 0 if date1 >= date2, 1 otherwise
compare_dates() {
    local date1="${1}"
    local date2="${2}"

    # Remove hyphens and compare as integers
    local num1="${date1//-/}"
    local num2="${date2//-/}"

    [ "${num1}" -ge "${num2}" ]
    return $?
}

# Validation
if [ ! -f "${AGENT_RULES_SOURCE}" ]; then
    print::error "could not locate agent rules source file: ${AGENT_RULES_SOURCE}"
    exit 1
fi

if [ ! -d "${CLAUDE_ROOT}" ]; then
    print::error "could not locate expected config for Claude Code: ${CLAUDE_ROOT}"
    exit 1
fi

# Create global CLAUDE.md if it doesn't exist
if [ ! -f "${GLOBAL_CONF}" ]; then
    create_global_claude_config || exit 1
fi

# Extract source date from agent rules
source_date=$(extract_rules_date "${AGENT_RULES_SOURCE}")

if [ -z "${source_date}" ]; then
    print::error "could not extract date from agent rules source: ${AGENT_RULES_SOURCE}"
    exit 1
fi

# Check if agent rules already exist in CLAUDE.md
if has_agent_rules "${GLOBAL_CONF}"; then
    # Extract installed date
    installed_date=$(extract_rules_date "${GLOBAL_CONF}")

    if [ -z "${installed_date}" ]; then
        print::warning "found agent rules but could not extract date, will reinstall"
    elif compare_dates "${installed_date}" "${source_date}"; then
        print::info "agent rules are up to date (${installed_date})"
        exit 0
    else
        print::info "updating agent rules (old: ${installed_date}, new: ${source_date})"
        backup_global_claude_config || exit 1
        remove_existing_agent_rules "${GLOBAL_CONF}" || exit 1
    fi
else
    print::info "installing agent rules for the first time (${source_date})"
    backup_global_claude_config || exit 1
fi

# Append agent rules to CLAUDE.md
cat "${AGENT_RULES_SOURCE}" >> "${GLOBAL_CONF}" || {
    print::error "failed to append agent rules to global config: ${GLOBAL_CONF}"
    exit 1
}

print::success "agent rules written to global CLAUDE.md (${source_date})"