#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
WRKBNCH_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Logging setup
TIMESTAMP="${TIMESTAMP:-$(date +%Y%m%d-%H%M%S)}"
LOG_DIR="${SCRIPT_DIR}/logs/${TIMESTAMP}"
LOG_FILE="${LOG_DIR}/02-install-skills.log"

mkdir -p "${LOG_DIR}"
exec > >(tee -a "${LOG_FILE}") 2>&1

printf "Logging to: %s\n" "${LOG_FILE}"

SKILL_SOURCE="${WRKBNCH_DIR}/../skills"
SKILLS_DIR="${SKILLS_DIR:-${HOME}/.claude/skills/}"
PROJECT_MARKER="team-agentic-setup"

validate_environment() {
    if [ -z "${SKILLS_DIR}" ]; then
        printf "ERROR: SKILLS_DIR is not set\n" >&2
        return 1
    fi

    if [ ! -d "${SKILL_SOURCE}" ]; then
        printf "ERROR: cannot locate the projects skill directory: %s\n" "${SKILL_SOURCE}" >&2
        return 1
    fi

    return 0
}

is_project_skill() {
    local skill_dir="${1}"

    if [ ! -d "${skill_dir}" ]; then
        return 1
    fi

    local skill_file="${skill_dir}/SKILL.md"

    if [ ! -f "${skill_file}" ]; then
        return 1
    fi

    # Extract only the YAML frontmatter (between first two --- delimiters)
    # Use awk to get only the first frontmatter block
    local frontmatter
    frontmatter="$(awk '/^---$/ {if (++count == 2) exit} count == 1 && NR > 1' "${skill_file}")"

    # Parse the frontmatter with yq
    local project_value
    project_value="$(printf '%s\n' "${frontmatter}" | yq eval '.project_skill' - 2>/dev/null)"

    if [ -z "${project_value}" ] || [ "${project_value}" = "null" ]; then
        return 1
    fi

    if [ "${project_value}" = "${PROJECT_MARKER}" ]; then
        return 0
    fi

    return 1
}

remove_project_skills() {
    local removed_count=0
    local preserved_count=0

    if [ ! -d "${SKILLS_DIR}" ]; then
        return 0
    fi

    printf "Scanning existing skills...\n"

    for skill_dir in "${SKILLS_DIR}"*/; do
        if [ ! -d "${skill_dir}" ]; then
            continue
        fi

        local skill_name
        skill_name="$(basename "${skill_dir}")"

        if is_project_skill "${skill_dir}"; then
            printf "  Removing project skill: %s\n" "${skill_name}"
            rm -rf "${skill_dir}"
            removed_count=$((removed_count + 1))
        else
            printf "  Preserving user skill: %s\n" "${skill_name}"
            preserved_count=$((preserved_count + 1))
        fi
    done

    printf "Removed %d project skill(s), preserved %d user skill(s)\n" "${removed_count}" "${preserved_count}"
    return 0
}

install_project_skills() {
    local installed_count=0
    local skipped_count=0

    printf "Installing project skills from %s...\n" "${SKILL_SOURCE}"

    while IFS= read -r source_dir; do
        local skill_name
        skill_name="$(basename "${source_dir}")"

        if is_project_skill "${source_dir}"; then
            cp -R "${source_dir}" "${SKILLS_DIR}${skill_name}"
            printf "  Installed: %s\n" "${skill_name}"
            installed_count=$((installed_count + 1))
        else
            printf "  Skipped (no project_skill metadata): %s\n" "${skill_name}"
            skipped_count=$((skipped_count + 1))
        fi
    done < <(find "${SKILL_SOURCE}" -mindepth 1 -maxdepth 1 -type d)

    printf "Installed %d project skill(s), skipped %d directory/directories\n" "${installed_count}" "${skipped_count}"
    return 0
}

install_skills() {
    if ! validate_environment; then
        return 1
    fi

    if [ ! -d "${SKILLS_DIR}" ]; then
        printf "Creating skills directory: %s\n" "${SKILLS_DIR}"
        mkdir -p "${SKILLS_DIR}"
    fi

    if ! remove_project_skills; then
        printf "ERROR: failed to remove project skills\n" >&2
        return 1
    fi

    if ! install_project_skills; then
        printf "ERROR: failed to install project skills\n" >&2
        return 1
    fi

    printf "\nSUCCESS: all skills updated\n"
    return 0
}

install_skills
