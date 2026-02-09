#!/usr/bin/env bash
# ====================
# Pattern Metadata Validation Script
# ====================
# Validates YAML frontmatter metadata in pattern markdown files

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Logging setup
TIMESTAMP="${TIMESTAMP:-$(date +%Y%m%d-%H%M%S)}"
LOG_DIR="${SCRIPT_DIR}/logs/${TIMESTAMP}"
LOG_FILE="${LOG_DIR}/04-validate-metadata.log"

mkdir -p "${LOG_DIR}"
exec > >(tee -a "${LOG_FILE}") 2>&1

printf "Logging to: %s\n" "${LOG_FILE}"

# Source print library
# shellcheck source=lib/print.sh
. "${SCRIPT_DIR}/../lib/print.sh"

# Configuration
PATTERNS_DIR="${PATTERNS_DIR:-${SCRIPT_DIR}/../../patterns}"

# No dependency check - yq and jq are prerequisites (see README)

# Extract YAML frontmatter from markdown file
extract_frontmatter() {
    local file="$1"

    # Extract content between --- markers
    awk '/^---$/{flag=!flag;next}flag' "$file"
}

# Parse metadata from frontmatter using yq
parse_metadata() {
    local frontmatter="$1"

    # Use yq to parse YAML and convert to JSON
    # Try mikefarah/yq syntax first, fall back to kislyuk/yq
    if printf "%s" "$frontmatter" | yq eval -o=json '.' 2>/dev/null; then
        return 0
    elif printf "%s" "$frontmatter" | yq -o=json '.' 2>/dev/null; then
        return 0
    else
        # kislyuk/yq (jq wrapper) - outputs JSON by default
        printf "%s" "$frontmatter" | yq '.'
    fi
}

# Validate required metadata fields
validate_metadata() {
    local metadata="$1"
    local file="$2"

    local required_fields=("entity_name" "entity_type" "language" "domain" "description")
    local missing=()

    for field in "${required_fields[@]}"; do
        if ! printf "%s" "$metadata" | jq -e ".$field" >/dev/null 2>&1; then
            missing+=("$field")
        fi
    done

    if [ ${#missing[@]} -gt 0 ]; then
        print::error "Missing required fields in $file: ${missing[*]}"
        return 1
    fi

    return 0
}

# Validate single pattern file
validate_pattern() {
    local file="$1"

    # Skip README files
    if [[ "$file" == */README.md ]]; then
        print::warning "README file found in $file, skipping"
        return 0
    fi

    print::info "Processing: $file"

    # Extract frontmatter
    local frontmatter
    frontmatter=$(extract_frontmatter "$file")

    if [ -z "$frontmatter" ]; then
        print::error "No frontmatter found in $file (frontmatter is required)"
        return 1
    fi

    # Parse metadata
    local metadata
    metadata=$(parse_metadata "$frontmatter")

    if ! validate_metadata "$metadata" "$file"; then
        return 1
    fi

    # Extract and display key metadata
    local entity_name
    local entity_type
    local language
    local domain
    local tags

    entity_name=$(printf "%s" "$metadata" | jq -r '.entity_name')
    entity_type=$(printf "%s" "$metadata" | jq -r '.entity_type')
    language=$(printf "%s" "$metadata" | jq -r '.language')
    domain=$(printf "%s" "$metadata" | jq -r '.domain')
    tags=$(printf "%s" "$metadata" | jq -c '.tags // []')

    print::success "Valid: $entity_name"
    print::info "  Type: $entity_type | Language: $language | Domain: $domain"
    print::info "  Tags: $tags"

    return 0
}

# Validate all patterns from directory
validate_patterns_from_dir() {
    local dir="$1"

    if [ ! -d "$dir" ]; then
        print::error "Directory not found: $dir"
        return 1
    fi

    local pattern_count=0
    local success_count=0
    local error_count=0

    # Find all markdown files
    while IFS= read -r -d '' file; do
        ((pattern_count++))

        if validate_pattern "$file"; then
            ((success_count++))
        else
            ((error_count++))
        fi
        echo ""  # Add blank line between patterns
    done < <(find "$dir" -name "*.md" -type f -print0 | sort -z)

    print::info "Summary:"
    print::info "  Total patterns: $pattern_count"
    print::success "  Successfully validated: $success_count"
    if [ $error_count -gt 0 ]; then
        print::error "  Errors: $error_count"
        return 1
    fi
}

# Main execution
main() {
    print::info "Pattern Metadata Validation Script"
    print::info "Scanning for patterns in: $PATTERNS_DIR"
    print::info ""

    if ! validate_patterns_from_dir "$PATTERNS_DIR"; then
        return 1
    fi

    print::info ""
    print::success "Validation complete!"
}

# Run main function
main "$@"
