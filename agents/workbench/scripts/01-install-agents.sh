#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
WRKBNCH_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Logging setup
TIMESTAMP="${TIMESTAMP:-$(date +%Y%m%d-%H%M%S)}"
LOG_DIR="${SCRIPT_DIR}/logs/${TIMESTAMP}"
LOG_FILE="${LOG_DIR}/01-install-agents.log"

mkdir -p "${LOG_DIR}"
exec > >(tee -a "${LOG_FILE}") 2>&1

printf "Logging to: %s\n" "${LOG_FILE}"

AGENT_SOURCE="${WRKBNCH_DIR}/../definitions"
AGENTS_DIR="${AGENTS_DIR:-${HOME}/.claude/agents/}"
PROJECT_MARKER="team-agentic-setup"

validate_environment() {
    if [ -z "${AGENTS_DIR}" ]; then
        printf "ERROR: AGENTS_DIR is not set\n" >&2
        return 1
    fi

    if [ ! -d "${AGENT_SOURCE}" ]; then
        printf "ERROR: cannot locate the projects agent definitions directory: %s\n" "${AGENT_SOURCE}" >&2
        return 1
    fi

    return 0
}

is_project_agent() {
    local agent_file="${1}"

    if [ ! -f "${agent_file}" ]; then
        return 1
    fi

    # Extract only the YAML frontmatter (between first two --- delimiters)
    # Use awk to get only the first frontmatter block
    local frontmatter
    frontmatter="$(awk '/^---$/ {if (++count == 2) exit} count == 1 && NR > 1' "${agent_file}")"

    # Parse the frontmatter with yq
    local project_value
    project_value="$(printf '%s\n' "${frontmatter}" | yq eval '.project_agent' - 2>/dev/null)"

    if [ -z "${project_value}" ] || [ "${project_value}" = "null" ]; then
        return 1
    fi

    if [ "${project_value}" = "${PROJECT_MARKER}" ]; then
        return 0
    fi

    return 1
}

remove_project_agents() {
    local removed_count=0
    local preserved_count=0

    if [ ! -d "${AGENTS_DIR}" ]; then
        return 0
    fi

    printf "Scanning existing agents...\n"

    for agent_file in "${AGENTS_DIR}"/*.md; do
        if [ ! -f "${agent_file}" ]; then
            continue
        fi

        local agent_name
        agent_name="$(basename "${agent_file}")"

        if is_project_agent "${agent_file}"; then
            printf "  Removing project agent: %s\n" "${agent_name}"
            rm -f "${agent_file}"
            removed_count=$((removed_count + 1))
        else
            printf "  Preserving user agent: %s\n" "${agent_name}"
            preserved_count=$((preserved_count + 1))
        fi
    done

    printf "Removed %d project agent(s), preserved %d user agent(s)\n" "${removed_count}" "${preserved_count}"
    return 0
}

install_project_agents() {
    local installed_count=0
    local skipped_count=0

    printf "Installing project agents from %s...\n" "${AGENT_SOURCE}"

    while IFS= read -r source_file; do
        local agent_name
        agent_name="$(basename "${source_file}")"

        if is_project_agent "${source_file}"; then
            cp "${source_file}" "${AGENTS_DIR}${agent_name}"
            printf "  Installed: %s\n" "${agent_name}"
            installed_count=$((installed_count + 1))
        else
            printf "  Skipped (no project_agent metadata): %s\n" "${agent_name}"
            skipped_count=$((skipped_count + 1))
        fi
    done < <(find "${AGENT_SOURCE}" -type f -name "*.md")

    printf "Installed %d project agent(s), skipped %d file(s)\n" "${installed_count}" "${skipped_count}"
    return 0
}

install_agents() {
    if ! validate_environment; then
        return 1
    fi

    if [ ! -d "${AGENTS_DIR}" ]; then
        printf "Creating agents directory: %s\n" "${AGENTS_DIR}"
        mkdir -p "${AGENTS_DIR}"
    fi

    if ! remove_project_agents; then
        printf "ERROR: failed to remove project agents\n" >&2
        return 1
    fi

    if ! install_project_agents; then
        printf "ERROR: failed to install project agents\n" >&2
        return 1
    fi

    printf "\nSUCCESS: all agents updated\n"
    return 0
}

install_agents
