#!/usr/bin/env bats

# Test suite for scripts/install-global-agent-rules.sh
# Tests the global agent rules installation script with date-aware behavior

# Global variable to store original CLAUDE.md content for restoration
ORIGINAL_CLAUDE_MD=""
ORIGINAL_CLAUDE_MD_EXISTS=false

# setup_file() runs once before all tests
setup_file() {
    # Save the original ~/.claude/CLAUDE.md if it exists
    if [ -f "${HOME}/.claude/CLAUDE.md" ]; then
        ORIGINAL_CLAUDE_MD_EXISTS=true
        ORIGINAL_CLAUDE_MD=$(cat "${HOME}/.claude/CLAUDE.md")
        export ORIGINAL_CLAUDE_MD
        export ORIGINAL_CLAUDE_MD_EXISTS
    else
        export ORIGINAL_CLAUDE_MD_EXISTS=false
    fi
}

# teardown_file() runs once after all tests complete
teardown_file() {
    # Restore the original ~/.claude/CLAUDE.md file
    if [ "$ORIGINAL_CLAUDE_MD_EXISTS" = "true" ]; then
        printf '%s\n' "$ORIGINAL_CLAUDE_MD" > "${HOME}/.claude/CLAUDE.md"
    else
        # Remove the file if it didn't exist originally
        rm -f "${HOME}/.claude/CLAUDE.md"
    fi

    # Clean up any backup files created during testing
    rm -f "${HOME}/.claude/CLAUDE.md.backup"
}

# setup() runs before each test
setup() {
    # Create isolated test environment
    export TEST_DIR="${BATS_TEST_TMPDIR}/test-$$"
    mkdir -p "$TEST_DIR"

    # Create fake CLAUDE_ROOT for testing
    export FAKE_CLAUDE_ROOT="${TEST_DIR}/claude"
    mkdir -p "$FAKE_CLAUDE_ROOT"

    # Create fake project structure
    export FAKE_PROJ_ROOT="${TEST_DIR}/project"
    export FAKE_AGENTS_DIR="${FAKE_PROJ_ROOT}/agents"
    export FAKE_SCRIPT_DIR="${FAKE_PROJ_ROOT}/scripts"
    export FAKE_LIB_DIR="${FAKE_SCRIPT_DIR}/lib"
    mkdir -p "$FAKE_AGENTS_DIR"
    mkdir -p "$FAKE_LIB_DIR"

    # Copy the print.sh library to fake lib directory
    cp "/Users/doublej/dev/mcp/team-agentic-setup/scripts/lib/print.sh" "$FAKE_LIB_DIR/"

    # Copy the actual script to test directory
    cp "/Users/doublej/dev/mcp/team-agentic-setup/scripts/install-global-agent-rules.sh" "${TEST_DIR}/original-script.sh"
}

# teardown() runs after each test
teardown() {
    # Clean up test directory
    rm -rf "$TEST_DIR"
}

# Helper function to create test agent rules file with specific date
create_agent_rules_file() {
    local date="${1}"
    cat > "${FAKE_AGENTS_DIR}/global-agent-rules.txt" << EOF
<!-- BEGIN AGENT RULES -->

## MANDATORY PRE-RESPONSE CHECK

**Last Updated: ${date}**

**Before responding to ANY user request, STOP and check this delegation table:**

### Task Type → Specialist Agent Mapping

| Task Keywords                      | Delegate To                 | Never Do Yourself        |
| ---------------------------------- | --------------------------- | ------------------------ |
| "write/create **BATS tests**"      | \`bats-test-agent\`         | Don't explore/plan tests |
| "write/create **shell script**"    | \`shell-script-agent\`      | Don't write .sh files    |

### Red Flags - If You're Doing These, STOP

- Creating TodoWrite with: "Explore", "Implement", "Write code"
- Using Read/Grep/Glob to explore code before delegating

<!-- END AGENT RULES -->
EOF
}

# Helper function to create CLAUDE.md with agent rules at specific date
create_claude_md_with_rules() {
    local date="${1}"
    cat > "${FAKE_CLAUDE_ROOT}/CLAUDE.md" << EOF
# CLAUDE.md

## Existing User Content

This is existing user content that should be preserved.

<!-- BEGIN AGENT RULES -->

## MANDATORY PRE-RESPONSE CHECK

**Last Updated: ${date}**

**Before responding to ANY user request, STOP and check this delegation table:**

### Task Type → Specialist Agent Mapping

| Task Keywords                      | Delegate To                 | Never Do Yourself        |
| ---------------------------------- | --------------------------- | ------------------------ |
| "write/create **BATS tests**"      | \`old-bats-agent\`          | Don't explore/plan tests |

<!-- END AGENT RULES -->

## More User Content

Additional user content after agent rules.
EOF
}

# Helper function to create wrapper script for testing
create_test_wrapper_script() {
    cat > "${TEST_DIR}/test-script.sh" << 'WRAPPER_EOF'
#!/usr/bin/env bash

set -e

SCRIPT_DIR="${FAKE_SCRIPT_DIR}"
PROJ_ROOT="${FAKE_PROJ_ROOT}"
CLAUDE_ROOT="${FAKE_CLAUDE_ROOT}"
GLOBAL_CONF="${CLAUDE_ROOT}/CLAUDE.md"
AGENT_RULES_SOURCE="${PROJ_ROOT}/agents/global-agent-rules.txt"

source "${SCRIPT_DIR}/lib/print.sh"

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

backup_global_claude_config(){
    local backup="${GLOBAL_CONF}.backup"

    cp "${GLOBAL_CONF}" "${backup}" || {
        print::error "failed to backup the global config: ${GLOBAL_CONF}"
        return 1
    }

    return 0
}

extract_rules_date() {
    local file="${1}"

    if [ ! -f "${file}" ]; then
        return 0
    fi

    grep -E '^\*\*Last Updated: [0-9]{4}-[0-9]{2}-[0-9]{2}\*\*$' "${file}" | \
        sed -E 's/^\*\*Last Updated: ([0-9]{4}-[0-9]{2}-[0-9]{2})\*\*$/\1/' | \
        head -n 1

    return 0
}

has_agent_rules() {
    local file="${1}"

    if [ ! -f "${file}" ]; then
        return 1
    fi

    grep -q "<!-- BEGIN AGENT RULES -->" "${file}"
    return $?
}

remove_existing_agent_rules() {
    local claude_file="${1}"

    if [ ! -f "${claude_file}" ]; then
        print::error "file does not exist: ${claude_file}"
        return 1
    fi

    local temp_file
    temp_file="$(mktemp)" || {
        print::error "failed to create temporary file"
        return 1
    }

    sed '/<!-- BEGIN AGENT RULES -->/,/<!-- END AGENT RULES -->/d' "${claude_file}" > "${temp_file}" || {
        rm -f "${temp_file}"
        print::error "failed to remove agent rules section"
        return 1
    }

    mv "${temp_file}" "${claude_file}" || {
        rm -f "${temp_file}"
        print::error "failed to update ${claude_file}"
        return 1
    }

    return 0
}

compare_dates() {
    local date1="${1}"
    local date2="${2}"

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
        backup_global_claude_config || exit 1
        remove_existing_agent_rules "${GLOBAL_CONF}" || exit 1
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
WRAPPER_EOF

    chmod +x "${TEST_DIR}/test-script.sh"
}

# ========================================
# Happy Path: First-time Installation
# ========================================

@test "installs rules on first run when none exist" {
    # Arrange: Create source rules with current date
    create_agent_rules_file "2025-11-14"

    # Create CLAUDE.md without agent rules
    cat > "${FAKE_CLAUDE_ROOT}/CLAUDE.md" << 'EOF'
# CLAUDE.md

## Existing User Section

This is existing user content.
EOF

    # Create test wrapper script
    create_test_wrapper_script

    # Act: Run the script
    run "${TEST_DIR}/test-script.sh"

    # Assert: Verify success
    [ "$status" -eq 0 ]

    # Verify success message appears
    printf '%s\n' "$output" | grep -q '\[INFO\] installing agent rules for the first time (2025-11-14)'
    printf '%s\n' "$output" | grep -q '\[SUCCESS\] agent rules written to global CLAUDE.md (2025-11-14)'

    # Verify backup was created
    [ -f "${FAKE_CLAUDE_ROOT}/CLAUDE.md.backup" ]

    # Verify user content is preserved
    grep -q "## Existing User Section" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"
    grep -q "This is existing user content." "${FAKE_CLAUDE_ROOT}/CLAUDE.md"

    # Verify agent rules were appended with HTML markers
    grep -q "<!-- BEGIN AGENT RULES -->" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"
    grep -q "<!-- END AGENT RULES -->" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"

    # Verify H2 heading (not H1)
    grep -q "^## MANDATORY PRE-RESPONSE CHECK$" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"

    # Verify date header
    grep -q "^\*\*Last Updated: 2025-11-14\*\*$" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"
}

# ========================================
# Idempotency: Same Date = No Update
# ========================================

@test "running script multiple times with same date does not duplicate content" {
    # Arrange: Create source rules with date 2025-11-14
    create_agent_rules_file "2025-11-14"

    # Create CLAUDE.md with rules at same date
    create_claude_md_with_rules "2025-11-14"

    # Create test wrapper script
    create_test_wrapper_script

    # Act: Run the script first time
    run "${TEST_DIR}/test-script.sh"
    [ "$status" -eq 0 ]

    # Verify it was up to date
    printf '%s\n' "$output" | grep -q '\[INFO\] agent rules are up to date (2025-11-14)'

    # Count occurrences of agent rules section
    local first_run_count
    first_run_count=$(grep -c "<!-- BEGIN AGENT RULES -->" "${FAKE_CLAUDE_ROOT}/CLAUDE.md")
    [ "$first_run_count" -eq 1 ]

    # Run the script second time
    run "${TEST_DIR}/test-script.sh"
    [ "$status" -eq 0 ]

    # Assert: Content should still appear exactly once
    local second_run_count
    second_run_count=$(grep -c "<!-- BEGIN AGENT RULES -->" "${FAKE_CLAUDE_ROOT}/CLAUDE.md")
    [ "$second_run_count" -eq 1 ]

    # Note: First run creates backup, second run doesn't (exits early with "up to date")
    # So backup file exists from first run, but wasn't created by second run
}

# ========================================
# Date Comparison: Skip Update When Up to Date
# ========================================

@test "skips update when installed rules are up to date" {
    # Arrange: Create source rules with date 2025-11-14
    create_agent_rules_file "2025-11-14"

    # Create CLAUDE.md with rules at same date
    create_claude_md_with_rules "2025-11-14"

    # Create test wrapper script
    create_test_wrapper_script

    # Save original content
    local original_content
    original_content=$(cat "${FAKE_CLAUDE_ROOT}/CLAUDE.md")

    # Act: Run the script
    run "${TEST_DIR}/test-script.sh"

    # Assert: Script exits successfully with up-to-date message
    [ "$status" -eq 0 ]
    printf '%s\n' "$output" | grep -q '\[INFO\] agent rules are up to date (2025-11-14)'

    # Verify content unchanged
    local current_content
    current_content=$(cat "${FAKE_CLAUDE_ROOT}/CLAUDE.md")
    [ "$original_content" = "$current_content" ]
}

@test "skips update when installed rules are newer than source" {
    # Arrange: Create source rules with older date
    create_agent_rules_file "2025-11-10"

    # Create CLAUDE.md with rules at newer date
    create_claude_md_with_rules "2025-11-14"

    # Create test wrapper script
    create_test_wrapper_script

    # Act: Run the script
    run "${TEST_DIR}/test-script.sh"

    # Assert: Script exits successfully with up-to-date message
    [ "$status" -eq 0 ]
    printf '%s\n' "$output" | grep -q '\[INFO\] agent rules are up to date (2025-11-14)'

    # Verify old content still present (not replaced)
    grep -q "old-bats-agent" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"
}

# ========================================
# Date Comparison: Update When Source is Newer
# ========================================

@test "updates when source rules are newer than installed" {
    # Arrange: Create source rules with newer date
    create_agent_rules_file "2025-11-20"

    # Create CLAUDE.md with rules at older date
    create_claude_md_with_rules "2025-11-10"

    # Create test wrapper script
    create_test_wrapper_script

    # Act: Run the script
    run "${TEST_DIR}/test-script.sh"

    # Assert: Verify update occurred
    [ "$status" -eq 0 ]
    printf '%s\n' "$output" | grep -q '\[INFO\] updating agent rules (old: 2025-11-10, new: 2025-11-20)'
    printf '%s\n' "$output" | grep -q '\[SUCCESS\] agent rules written to global CLAUDE.md (2025-11-20)'

    # Verify backup was created
    [ -f "${FAKE_CLAUDE_ROOT}/CLAUDE.md.backup" ]

    # Verify backup contains old date
    grep -q "^\*\*Last Updated: 2025-11-10\*\*$" "${FAKE_CLAUDE_ROOT}/CLAUDE.md.backup"

    # Verify new date in updated file
    grep -q "^\*\*Last Updated: 2025-11-20\*\*$" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"

    # Verify new content is present
    grep -q "bats-test-agent" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"

    # Verify old content was removed
    run grep -q "old-bats-agent" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"
    [ "$status" -ne 0 ]
}

# ========================================
# User Content Preservation
# ========================================

@test "preserves user content when updating rules" {
    # Arrange: Create source rules with newer date
    create_agent_rules_file "2025-11-20"

    # Create CLAUDE.md with rules at older date and user content
    create_claude_md_with_rules "2025-11-10"

    # Create test wrapper script
    create_test_wrapper_script

    # Act: Run the script
    run "${TEST_DIR}/test-script.sh"

    # Assert: Verify success
    [ "$status" -eq 0 ]

    # Verify user content before agent rules is preserved
    grep -q "## Existing User Content" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"
    grep -q "This is existing user content that should be preserved." "${FAKE_CLAUDE_ROOT}/CLAUDE.md"

    # Verify user content after agent rules is preserved
    grep -q "## More User Content" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"
    grep -q "Additional user content after agent rules." "${FAKE_CLAUDE_ROOT}/CLAUDE.md"

    # Verify agent rules were updated (new date)
    grep -q "^\*\*Last Updated: 2025-11-20\*\*$" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"

    # Verify only one agent rules section exists
    local section_count
    section_count=$(grep -c "<!-- BEGIN AGENT RULES -->" "${FAKE_CLAUDE_ROOT}/CLAUDE.md")
    [ "$section_count" -eq 1 ]
}

# ========================================
# Date Extraction Tests
# ========================================

@test "extracts date correctly from rules file" {
    # Arrange: Create source rules with specific date
    create_agent_rules_file "2025-12-25"

    # Create test wrapper script
    create_test_wrapper_script

    # Create minimal CLAUDE.md
    printf "# CLAUDE.md\n" > "${FAKE_CLAUDE_ROOT}/CLAUDE.md"

    # Act: Run the script
    run "${TEST_DIR}/test-script.sh"

    # Assert: Verify correct date was extracted and used
    [ "$status" -eq 0 ]
    printf '%s\n' "$output" | grep -q '2025-12-25'

    # Verify date appears in final file
    grep -q "^\*\*Last Updated: 2025-12-25\*\*$" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"
}

# ========================================
# Error Handling: Missing Source File
# ========================================

@test "fails when agent rules source file does not exist" {
    # Arrange: Do not create agent rules file
    # Create test wrapper script
    create_test_wrapper_script

    # Act: Run the script
    run "${TEST_DIR}/test-script.sh"

    # Assert: Verify failure
    [ "$status" -eq 1 ]

    # Verify error message appears
    printf '%s\n' "$output" | grep -q '\[ERROR\] could not locate agent rules source file'
}

# ========================================
# Error Handling: Missing CLAUDE_ROOT
# ========================================

@test "fails when CLAUDE_ROOT directory does not exist" {
    # Arrange: Create source rules
    create_agent_rules_file "2025-11-14"

    # Remove CLAUDE_ROOT directory
    rm -rf "$FAKE_CLAUDE_ROOT"

    # Create test wrapper script
    create_test_wrapper_script

    # Act: Run the script
    run "${TEST_DIR}/test-script.sh"

    # Assert: Verify failure
    [ "$status" -eq 1 ]

    # Verify error message appears
    printf '%s\n' "$output" | grep -q '\[ERROR\] could not locate expected config for Claude Code'
}

# ========================================
# Error Handling: Missing Date in Source
# ========================================

@test "fails when source rules file has no date" {
    # Arrange: Create source rules WITHOUT date header
    cat > "${FAKE_AGENTS_DIR}/global-agent-rules.txt" << 'EOF'
<!-- BEGIN AGENT RULES -->

## MANDATORY PRE-RESPONSE CHECK

**Before responding to ANY user request, STOP and check this delegation table:**

<!-- END AGENT RULES -->
EOF

    # Create test wrapper script
    create_test_wrapper_script

    # Create minimal CLAUDE.md
    printf "# CLAUDE.md\n" > "${FAKE_CLAUDE_ROOT}/CLAUDE.md"

    # Act: Run the script
    run "${TEST_DIR}/test-script.sh"

    # Assert: Verify failure
    [ "$status" -eq 1 ]

    # Verify error message appears
    printf '%s\n' "$output" | grep -q '\[ERROR\] could not extract date from agent rules source'
}

# ========================================
# Error Handling: Missing Date in Installed Rules
# ========================================

@test "reinstalls when installed rules have no date" {
    # Arrange: Create source rules with date
    create_agent_rules_file "2025-11-14"

    # Create CLAUDE.md with rules but NO date
    cat > "${FAKE_CLAUDE_ROOT}/CLAUDE.md" << 'EOF'
# CLAUDE.md

## Existing Content

<!-- BEGIN AGENT RULES -->

## MANDATORY PRE-RESPONSE CHECK

**Before responding to ANY user request, STOP and check this delegation table:**

<!-- END AGENT RULES -->
EOF

    # Create test wrapper script
    create_test_wrapper_script

    # Act: Run the script
    run "${TEST_DIR}/test-script.sh"

    # Assert: Verify warning and reinstall
    [ "$status" -eq 0 ]
    printf '%s\n' "$output" | grep -q '\[WARNING\] found agent rules but could not extract date, will reinstall'
    printf '%s\n' "$output" | grep -q '\[SUCCESS\] agent rules written to global CLAUDE.md (2025-11-14)'

    # Verify new rules with date were installed
    grep -q "^\*\*Last Updated: 2025-11-14\*\*$" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"
}

# ========================================
# File Creation: CLAUDE.md Doesn't Exist
# ========================================

@test "creates CLAUDE.md when it does not exist" {
    # Arrange: Create source rules
    create_agent_rules_file "2025-11-14"

    # Do not create CLAUDE.md (it doesn't exist)

    # Create test wrapper script
    create_test_wrapper_script

    # Act: Run the script
    run "${TEST_DIR}/test-script.sh"

    # Assert: Verify success
    [ "$status" -eq 0 ]

    # Verify creation message appears
    printf '%s\n' "$output" | grep -q '\[INFO\] no global CLAUDE.md file exists - creating'
    printf '%s\n' "$output" | grep -q '\[SUCCESS\] global CLAUDE.md file created'

    # Verify file was created
    [ -f "${FAKE_CLAUDE_ROOT}/CLAUDE.md" ]

    # Verify header was written
    grep -q "# CLAUDE.md" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"

    # Verify agent rules were installed
    grep -q "<!-- BEGIN AGENT RULES -->" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"
    grep -q "^\*\*Last Updated: 2025-11-14\*\*$" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"
}

# ========================================
# Backup Behavior
# ========================================

@test "creates backup before updating existing file" {
    # Arrange: Create source rules
    create_agent_rules_file "2025-11-14"

    # Create existing CLAUDE.md without agent rules
    cat > "${FAKE_CLAUDE_ROOT}/CLAUDE.md" << 'EOF'
# CLAUDE.md

## Original Content

This should be backed up.
EOF

    # Create test wrapper script
    create_test_wrapper_script

    # Act: Run the script
    run "${TEST_DIR}/test-script.sh"

    # Assert: Verify success
    [ "$status" -eq 0 ]

    # Verify backup file was created
    [ -f "${FAKE_CLAUDE_ROOT}/CLAUDE.md.backup" ]

    # Verify backup contains original content
    grep -q "## Original Content" "${FAKE_CLAUDE_ROOT}/CLAUDE.md.backup"
    grep -q "This should be backed up." "${FAKE_CLAUDE_ROOT}/CLAUDE.md.backup"

    # Verify backup does NOT contain new agent rules
    run grep -q "<!-- BEGIN AGENT RULES -->" "${FAKE_CLAUDE_ROOT}/CLAUDE.md.backup"
    [ "$status" -ne 0 ]
}

@test "creates backup even when installing for first time" {
    # Arrange: Create source rules
    create_agent_rules_file "2025-11-14"

    # Do not create CLAUDE.md

    # Create test wrapper script
    create_test_wrapper_script

    # Act: Run the script
    run "${TEST_DIR}/test-script.sh"

    # Assert: Verify success
    [ "$status" -eq 0 ]

    # Verify backup WAS created (script creates backup at line 279 for first-time install)
    [ -f "${FAKE_CLAUDE_ROOT}/CLAUDE.md.backup" ]

    # Verify CLAUDE.md was created with agent rules
    [ -f "${FAKE_CLAUDE_ROOT}/CLAUDE.md" ]
    grep -q "<!-- BEGIN AGENT RULES -->" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"

    # Backup should contain just the header (created before appending rules)
    grep -q "# CLAUDE.md" "${FAKE_CLAUDE_ROOT}/CLAUDE.md.backup"
    run grep -q "<!-- BEGIN AGENT RULES -->" "${FAKE_CLAUDE_ROOT}/CLAUDE.md.backup"
    [ "$status" -ne 0 ]
}

# ========================================
# Section Removal Tests
# ========================================

@test "removes old section before installing new when updating" {
    # Arrange: Create source rules with newer date
    create_agent_rules_file "2025-11-20"

    # Create CLAUDE.md with old agent rules
    create_claude_md_with_rules "2025-11-10"

    # Create test wrapper script
    create_test_wrapper_script

    # Save initial agent rules section count
    local initial_count
    initial_count=$(grep -c "<!-- BEGIN AGENT RULES -->" "${FAKE_CLAUDE_ROOT}/CLAUDE.md")
    [ "$initial_count" -eq 1 ]

    # Act: Run the script
    run "${TEST_DIR}/test-script.sh"

    # Assert: Verify success
    [ "$status" -eq 0 ]

    # Verify only ONE agent rules section exists (old was removed, new was added)
    local final_count
    final_count=$(grep -c "<!-- BEGIN AGENT RULES -->" "${FAKE_CLAUDE_ROOT}/CLAUDE.md")
    [ "$final_count" -eq 1 ]

    # Verify old date is gone
    run grep -q "^\*\*Last Updated: 2025-11-10\*\*$" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"
    [ "$status" -ne 0 ]

    # Verify new date is present
    grep -q "^\*\*Last Updated: 2025-11-20\*\*$" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"
}

# ========================================
# Content Format Verification
# ========================================

@test "installed rules use HTML comment markers not plain text" {
    # Arrange: Create source rules
    create_agent_rules_file "2025-11-14"

    # Create minimal CLAUDE.md
    printf "# CLAUDE.md\n" > "${FAKE_CLAUDE_ROOT}/CLAUDE.md"

    # Create test wrapper script
    create_test_wrapper_script

    # Act: Run the script
    run "${TEST_DIR}/test-script.sh"

    # Assert: Verify HTML comment markers are used
    grep -q "<!-- BEGIN AGENT RULES -->" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"
    grep -q "<!-- END AGENT RULES -->" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"

    # Verify plain text markers are NOT used
    run grep -q "==== BEGIN AGENT RULES ====" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"
    [ "$status" -ne 0 ]
    run grep -q "==== END AGENT RULES ====" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"
    [ "$status" -ne 0 ]
}

@test "installed rules use H2 heading not H1" {
    # Arrange: Create source rules
    create_agent_rules_file "2025-11-14"

    # Create minimal CLAUDE.md
    printf "# CLAUDE.md\n" > "${FAKE_CLAUDE_ROOT}/CLAUDE.md"

    # Create test wrapper script
    create_test_wrapper_script

    # Act: Run the script
    run "${TEST_DIR}/test-script.sh"

    # Assert: Verify H2 heading is used (## not #)
    grep -q "^## MANDATORY PRE-RESPONSE CHECK$" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"

    # Verify H1 heading is NOT used for agent rules section
    run grep -q "^# MANDATORY PRE-RESPONSE CHECK$" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"
    [ "$status" -ne 0 ]
}

@test "installed rules include date header in correct format" {
    # Arrange: Create source rules
    create_agent_rules_file "2025-11-14"

    # Create minimal CLAUDE.md
    printf "# CLAUDE.md\n" > "${FAKE_CLAUDE_ROOT}/CLAUDE.md"

    # Create test wrapper script
    create_test_wrapper_script

    # Act: Run the script
    run "${TEST_DIR}/test-script.sh"

    # Assert: Verify date header in correct format
    grep -q "^\*\*Last Updated: 2025-11-14\*\*$" "${FAKE_CLAUDE_ROOT}/CLAUDE.md"

    # Verify date appears between BEGIN marker and main heading
    local content
    content=$(cat "${FAKE_CLAUDE_ROOT}/CLAUDE.md")

    # Extract agent rules section
    local rules_section
    rules_section=$(printf '%s\n' "$content" | \
        sed -n '/<!-- BEGIN AGENT RULES -->/,/<!-- END AGENT RULES -->/p')

    # Verify structure: BEGIN -> blank line -> H2 -> blank line -> date
    printf '%s\n' "$rules_section" | grep -A 4 "<!-- BEGIN AGENT RULES -->" | \
        grep -q "^\*\*Last Updated: 2025-11-14\*\*$"
}
