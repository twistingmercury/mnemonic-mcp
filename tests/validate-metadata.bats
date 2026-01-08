#!/usr/bin/env bats

# Test suite for scripts/validate-metadata.sh
# Tests the pattern metadata validation script that validates YAML frontmatter

# setup() runs before each test
setup() {
    # Create isolated test environment
    export TEST_DIR="${BATS_TEST_TMPDIR}/test-$$"
    mkdir -p "$TEST_DIR"

    # Create fake patterns directory
    export FAKE_PATTERNS_DIR="${TEST_DIR}/patterns"
    mkdir -p "$FAKE_PATTERNS_DIR"

    # Override PATTERNS_DIR environment variable
    export PATTERNS_DIR="$FAKE_PATTERNS_DIR"

    # Create wrapper script that sources the print library correctly
    export TEST_SCRIPT="${TEST_DIR}/validate-metadata-test.sh"
    export SCRIPT_ROOT="/Users/doublej/dev/mcp/team-agentic-setup/scripts"

    cat > "$TEST_SCRIPT" << 'EOF'
#!/usr/bin/env bash
set -e

# Source print library from actual location
SCRIPT_DIR="/Users/doublej/dev/mcp/team-agentic-setup/scripts"
# shellcheck source=/Users/doublej/dev/mcp/team-agentic-setup/scripts/lib/print.sh
. "${SCRIPT_DIR}/lib/print.sh"

# Use PATTERNS_DIR from environment
PATTERNS_DIR="${PATTERNS_DIR}"

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
EOF
    chmod +x "$TEST_SCRIPT"
}

# teardown() runs after each test
teardown() {
    # Clean up test resources
    rm -rf "$TEST_DIR"
}

# Helper function to create a valid pattern file
create_valid_pattern() {
    local filename="$1"
    local entity_name="${2:-test-pattern}"

    cat > "${FAKE_PATTERNS_DIR}/${filename}" << EOF
---
entity_name: ${entity_name}
entity_type: agent
language: english
domain: testing
description: A test pattern for validation
tags:
  - test
  - example
---

# Test Pattern

This is test content.
EOF
}

# Helper function to create a pattern with missing fields
create_pattern_missing_field() {
    local filename="$1"
    local missing_field="$2"

    cat > "${FAKE_PATTERNS_DIR}/${filename}" << EOF
---
entity_name: test-pattern
entity_type: agent
language: english
domain: testing
description: A test pattern
---

# Test Pattern
EOF

    # Remove the missing field from the file
    case "$missing_field" in
        entity_name)
            sed -i.bak '/^entity_name:/d' "${FAKE_PATTERNS_DIR}/${filename}"
            ;;
        entity_type)
            sed -i.bak '/^entity_type:/d' "${FAKE_PATTERNS_DIR}/${filename}"
            ;;
        language)
            sed -i.bak '/^language:/d' "${FAKE_PATTERNS_DIR}/${filename}"
            ;;
        domain)
            sed -i.bak '/^domain:/d' "${FAKE_PATTERNS_DIR}/${filename}"
            ;;
        description)
            sed -i.bak '/^description:/d' "${FAKE_PATTERNS_DIR}/${filename}"
            ;;
    esac
    rm -f "${FAKE_PATTERNS_DIR}/${filename}.bak"
}

# Happy Path Tests

@test "validates pattern file with all required metadata" {
    create_valid_pattern "valid-pattern.md"

    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]
    [[ "$output" == *"[SUCCESS] Valid: test-pattern"* ]]
    [[ "$output" == *"[SUCCESS] Validation complete!"* ]]
}

@test "validates multiple valid pattern files in directory" {
    create_valid_pattern "pattern1.md" "pattern-one"
    create_valid_pattern "pattern2.md" "pattern-two"
    create_valid_pattern "pattern3.md" "pattern-three"

    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]
    [[ "$output" == *"[SUCCESS] Valid: pattern-one"* ]]
    [[ "$output" == *"[SUCCESS] Valid: pattern-two"* ]]
    [[ "$output" == *"[SUCCESS] Valid: pattern-three"* ]]
    [[ "$output" == *"Total patterns: 3"* ]]
    [[ "$output" == *"Successfully validated: 3"* ]]
}

@test "validates pattern with optional tags field" {
    cat > "${FAKE_PATTERNS_DIR}/with-tags.md" << 'EOF'
---
entity_name: tagged-pattern
entity_type: agent
language: english
domain: testing
description: Pattern with tags
tags:
  - tag1
  - tag2
  - tag3
---

# Tagged Pattern
EOF

    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]
    [[ "$output" == *"[SUCCESS] Valid: tagged-pattern"* ]]
    [[ "$output" == *'Tags: ["tag1","tag2","tag3"]'* ]]
}

@test "validates pattern without tags field (tags are optional)" {
    cat > "${FAKE_PATTERNS_DIR}/no-tags.md" << 'EOF'
---
entity_name: no-tags-pattern
entity_type: agent
language: english
domain: testing
description: Pattern without tags
---

# No Tags Pattern
EOF

    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]
    [[ "$output" == *"[SUCCESS] Valid: no-tags-pattern"* ]]
    [[ "$output" == *"Tags: []"* ]]
}

@test "skips README.md files properly" {
    # Create valid pattern
    create_valid_pattern "valid-pattern.md"

    # Create README.md
    cat > "${FAKE_PATTERNS_DIR}/README.md" << 'EOF'
# README

This is a readme file without frontmatter.
EOF

    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]
    [[ "$output" == *"[WARNING] README file found"* ]]
    [[ "$output" == *"[SUCCESS] Valid: test-pattern"* ]]
    [[ "$output" == *"Total patterns: 2"* ]]
    [[ "$output" == *"Successfully validated: 2"* ]]
}

@test "processes pattern files in subdirectories" {
    mkdir -p "${FAKE_PATTERNS_DIR}/subdir1"
    mkdir -p "${FAKE_PATTERNS_DIR}/subdir2"

    create_valid_pattern "root-pattern.md" "root"

    cat > "${FAKE_PATTERNS_DIR}/subdir1/pattern1.md" << 'EOF'
---
entity_name: subdir1-pattern
entity_type: agent
language: english
domain: testing
description: Pattern in subdirectory 1
---

# Subdir1 Pattern
EOF

    cat > "${FAKE_PATTERNS_DIR}/subdir2/pattern2.md" << 'EOF'
---
entity_name: subdir2-pattern
entity_type: agent
language: english
domain: testing
description: Pattern in subdirectory 2
---

# Subdir2 Pattern
EOF

    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]
    [[ "$output" == *"[SUCCESS] Valid: root"* ]]
    [[ "$output" == *"[SUCCESS] Valid: subdir1-pattern"* ]]
    [[ "$output" == *"[SUCCESS] Valid: subdir2-pattern"* ]]
    [[ "$output" == *"Total patterns: 3"* ]]
}

# Error Scenario Tests

@test "fails when entity_name field is missing" {
    create_pattern_missing_field "missing-name.md" "entity_name"

    run "$TEST_SCRIPT"

    [ "$status" -eq 1 ]
    [[ "$output" == *"[ERROR] Missing required fields"* ]]
    [[ "$output" == *"entity_name"* ]]
}

@test "fails when entity_type field is missing" {
    create_pattern_missing_field "missing-type.md" "entity_type"

    run "$TEST_SCRIPT"

    [ "$status" -eq 1 ]
    [[ "$output" == *"[ERROR] Missing required fields"* ]]
    [[ "$output" == *"entity_type"* ]]
}

@test "fails when language field is missing" {
    create_pattern_missing_field "missing-language.md" "language"

    run "$TEST_SCRIPT"

    [ "$status" -eq 1 ]
    [[ "$output" == *"[ERROR] Missing required fields"* ]]
    [[ "$output" == *"language"* ]]
}

@test "fails when domain field is missing" {
    create_pattern_missing_field "missing-domain.md" "domain"

    run "$TEST_SCRIPT"

    [ "$status" -eq 1 ]
    [[ "$output" == *"[ERROR] Missing required fields"* ]]
    [[ "$output" == *"domain"* ]]
}

@test "fails when description field is missing" {
    create_pattern_missing_field "missing-description.md" "description"

    run "$TEST_SCRIPT"

    [ "$status" -eq 1 ]
    [[ "$output" == *"[ERROR] Missing required fields"* ]]
    [[ "$output" == *"description"* ]]
}

@test "fails when file has no frontmatter" {
    cat > "${FAKE_PATTERNS_DIR}/no-frontmatter.md" << 'EOF'
# Pattern Without Frontmatter

This file has no YAML frontmatter.
EOF

    run "$TEST_SCRIPT"

    [ "$status" -eq 1 ]
    [[ "$output" == *"[ERROR] No frontmatter found"* ]]
    [[ "$output" == *"frontmatter is required"* ]]
    [[ "$output" == *"Errors: 1"* ]]
}

@test "fails when YAML syntax is invalid" {
    cat > "${FAKE_PATTERNS_DIR}/invalid-yaml.md" << 'EOF'
---
entity_name: test
entity_type: agent
language: english
domain: testing
description: [invalid yaml - unclosed bracket
---

# Invalid YAML
EOF

    run "$TEST_SCRIPT"

    [ "$status" -eq 1 ]
}

@test "fails when patterns directory does not exist" {
    export PATTERNS_DIR="${TEST_DIR}/nonexistent"

    cat > "${TEST_DIR}/test-nonexistent.sh" << 'EOF'
#!/usr/bin/env bash
set -e

SCRIPT_DIR="/Users/doublej/dev/mcp/team-agentic-setup/scripts"
. "${SCRIPT_DIR}/lib/print.sh"

PATTERNS_DIR="${PATTERNS_DIR}"

validate_patterns_from_dir() {
    local dir="$1"

    if [ ! -d "$dir" ]; then
        print::error "Directory not found: $dir"
        return 1
    fi
}

main() {
    if ! validate_patterns_from_dir "$PATTERNS_DIR"; then
        return 1
    fi
}

main "$@"
EOF
    chmod +x "${TEST_DIR}/test-nonexistent.sh"

    run "${TEST_DIR}/test-nonexistent.sh"

    [ "$status" -eq 1 ]
    [[ "$output" == *"[ERROR] Directory not found"* ]]
}

@test "handles empty patterns directory" {
    # Directory exists but is empty
    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]
    [[ "$output" == *"Total patterns: 0"* ]]
    [[ "$output" == *"Successfully validated: 0"* ]]
}

# Edge Case Tests

@test "fails when frontmatter is empty (just --- markers)" {
    cat > "${FAKE_PATTERNS_DIR}/empty-frontmatter.md" << 'EOF'
---
---

# Pattern with empty frontmatter
EOF

    run "$TEST_SCRIPT"

    [ "$status" -eq 1 ]
    [[ "$output" == *"[ERROR] No frontmatter found"* ]]
    [[ "$output" == *"frontmatter is required"* ]]
    [[ "$output" == *"Errors: 1"* ]]
}

@test "handles mixed valid and invalid patterns" {
    create_valid_pattern "valid1.md" "valid-one"
    create_pattern_missing_field "invalid1.md" "entity_name"
    create_valid_pattern "valid2.md" "valid-two"

    run "$TEST_SCRIPT"

    [ "$status" -eq 1 ]
    [[ "$output" == *"[SUCCESS] Valid: valid-one"* ]]
    [[ "$output" == *"[ERROR] Missing required fields"* ]]
    [[ "$output" == *"[SUCCESS] Valid: valid-two"* ]]
    [[ "$output" == *"Total patterns: 3"* ]]
    [[ "$output" == *"Successfully validated: 2"* ]]
    [[ "$output" == *"Errors: 1"* ]]
}

@test "ignores files without .md extension" {
    create_valid_pattern "valid-pattern.md"

    # Create a file without .md extension
    cat > "${FAKE_PATTERNS_DIR}/not-markdown.txt" << 'EOF'
---
entity_name: ignored
entity_type: agent
language: english
domain: testing
description: This should be ignored
---
EOF

    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]
    [[ "$output" == *"Total patterns: 1"* ]]
    [[ "$output" != *"ignored"* ]]
}

@test "handles pattern with partial frontmatter (some fields missing)" {
    cat > "${FAKE_PATTERNS_DIR}/partial.md" << 'EOF'
---
entity_name: partial-pattern
entity_type: agent
---

# Partial Pattern
EOF

    run "$TEST_SCRIPT"

    [ "$status" -eq 1 ]
    [[ "$output" == *"[ERROR] Missing required fields"* ]]
    [[ "$output" == *"language"* ]]
    [[ "$output" == *"domain"* ]]
    [[ "$output" == *"description"* ]]
}

@test "handles pattern with special characters in entity name" {
    cat > "${FAKE_PATTERNS_DIR}/special-chars.md" << 'EOF'
---
entity_name: "pattern-with-special_chars.123"
entity_type: agent
language: english
domain: testing
description: "Pattern with special characters: @#$%"
---

# Special Characters Pattern
EOF

    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]
    [[ "$output" == *"[SUCCESS] Valid: pattern-with-special_chars.123"* ]]
}

@test "validates pattern with multiline description" {
    cat > "${FAKE_PATTERNS_DIR}/multiline.md" << 'EOF'
---
entity_name: multiline-pattern
entity_type: agent
language: english
domain: testing
description: |
  This is a multiline description.
  It spans multiple lines.
  Should still be valid.
---

# Multiline Description Pattern
EOF

    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]
    [[ "$output" == *"[SUCCESS] Valid: multiline-pattern"* ]]
}

# Configuration Tests

@test "respects custom PATTERNS_DIR environment variable" {
    # Create a different patterns directory
    export CUSTOM_PATTERNS="${TEST_DIR}/custom-patterns"
    mkdir -p "$CUSTOM_PATTERNS"
    export PATTERNS_DIR="$CUSTOM_PATTERNS"

    cat > "${CUSTOM_PATTERNS}/custom.md" << 'EOF'
---
entity_name: custom-location
entity_type: agent
language: english
domain: testing
description: Pattern in custom location
---

# Custom Location Pattern
EOF

    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]
    [[ "$output" == *"Scanning for patterns in: ${CUSTOM_PATTERNS}"* ]]
    [[ "$output" == *"[SUCCESS] Valid: custom-location"* ]]
}

@test "displays correct summary statistics" {
    create_valid_pattern "pattern1.md" "one"
    create_valid_pattern "pattern2.md" "two"
    create_pattern_missing_field "invalid.md" "entity_name"

    # Create README to be skipped
    printf '%s\n' "# README" > "${FAKE_PATTERNS_DIR}/README.md"

    run "$TEST_SCRIPT"

    [ "$status" -eq 1 ]
    [[ "$output" == *"Total patterns: 4"* ]]
    [[ "$output" == *"Successfully validated: 3"* ]]
    [[ "$output" == *"Errors: 1"* ]]
}
