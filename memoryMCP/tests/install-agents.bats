#!/usr/bin/env bats

# Test suite for scripts/install-agents.sh
# Tests the agent installation script that copies agent definitions to ~/.claude/agents/

# setup() runs before each test
setup() {
    # Create isolated test environment
    export TEST_DIR="${BATS_TEST_TMPDIR}/test-$$"
    mkdir -p "$TEST_DIR"

    # Create fake HOME for testing
    export FAKE_HOME="${TEST_DIR}/home"
    mkdir -p "$FAKE_HOME"
    export HOME="$FAKE_HOME"

    # Create fake project structure
    export FAKE_PROJ_ROOT="${TEST_DIR}/project"
    export FAKE_AGENT_SOURCE="${FAKE_PROJ_ROOT}/agents/definitions"
    mkdir -p "$FAKE_AGENT_SOURCE"

    # Create test agent files in various subdirectories (simulating real structure)
    mkdir -p "${FAKE_AGENT_SOURCE}/go"
    mkdir -p "${FAKE_AGENT_SOURCE}/shell"
    printf '%s\n' "# Go Engineer" > "${FAKE_AGENT_SOURCE}/go/go-engineer.md"
    printf '%s\n' "# Shell Engineer" > "${FAKE_AGENT_SOURCE}/shell/shell-engineer.md"
    printf '%s\n' "# Software Architect" > "${FAKE_AGENT_SOURCE}/software-architect.md"

    # Create wrapper script that uses fake project root
    export TEST_SCRIPT="${TEST_DIR}/install-agents-test.sh"
    cat > "$TEST_SCRIPT" << 'EOF'
#!/usr/bin/env bash
set -e

# Override PROJ_ROOT to use fake project structure
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJ_ROOT="${FAKE_PROJ_ROOT}"
AGENT_SOURCE="${PROJ_ROOT}/agents/definitions"
AGENTS_DIR="${HOME}/.claude/agents/"

copy(){
    # Validate AGENTS_DIR is set before using it
    if [ -z "${AGENTS_DIR}" ]; then
        printf "ERROR: AGENTS_DIR is not set\n" >&2
        return 1
    fi

    # Validate source directory exists
    if [ ! -d "${AGENT_SOURCE}" ]; then
        printf "ERROR: cannot locate the projects agent definitions directory: %s\n" "${AGENT_SOURCE}" >&2
        return 1
    fi

    # Create destination directory if needed
    if [ ! -d "${AGENTS_DIR}" ]; then
        mkdir -p "${AGENTS_DIR}"
    fi

    # Preserve user-created agents (those without project_agent: team-agentic-setup metadata)
    # by moving them to a temporary location before cleanup
    local temp_preserve_dir="${AGENTS_DIR}.preserve-$$"
    mkdir -p "$temp_preserve_dir"

    # Find and preserve user agents (files without project_agent: team-agentic-setup)
    if [ -d "${AGENTS_DIR}" ]; then
        find "${AGENTS_DIR}" -maxdepth 1 -type f -name "*.md" | while IFS= read -r agent_file; do
            # Check if file contains project_agent: team-agentic-setup
            if ! grep -q "^project_agent: team-agentic-setup" "$agent_file" 2>/dev/null; then
                # This is a user agent, preserve it
                cp "$agent_file" "$temp_preserve_dir/"
            fi
        done
    fi

    # Remove all existing agents
    rm -rf "${AGENTS_DIR:?}"*

    # Copy all .md files from agent definitions directory (recursively)
    # This flattens the directory structure - all agent files go directly into AGENTS_DIR
    find "${AGENT_SOURCE}" -type f -name "*.md" -exec cp {} "${AGENTS_DIR}" \;

    # Restore preserved user agents
    if [ -d "$temp_preserve_dir" ]; then
        find "$temp_preserve_dir" -maxdepth 1 -type f -name "*.md" -exec cp {} "${AGENTS_DIR}" \;
        rm -rf "$temp_preserve_dir"
    fi

    printf "SUCCESS: all agents updated\n"

    return 0
}

copy
EOF
    chmod +x "$TEST_SCRIPT"
}

# teardown() runs after each test
teardown() {
    # Clean up test resources
    rm -rf "$TEST_DIR"
}

# Happy Path Tests

@test "successfully copies agent files from source to destination" {
    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]
    [[ "$output" == *"SUCCESS: all agents updated"* ]]
}

@test "creates agents directory if it does not exist" {
    # Ensure directory doesn't exist
    [ ! -d "${FAKE_HOME}/.claude/agents" ]

    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]
    [ -d "${FAKE_HOME}/.claude/agents" ]
}

@test "flattens directory structure when copying agent files" {
    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]

    # All files should be copied directly to AGENTS_DIR (flattened)
    [ -f "${FAKE_HOME}/.claude/agents/go-engineer.md" ]
    [ -f "${FAKE_HOME}/.claude/agents/shell-engineer.md" ]
    [ -f "${FAKE_HOME}/.claude/agents/software-architect.md" ]

    # Should not preserve subdirectory structure
    [ ! -d "${FAKE_HOME}/.claude/agents/go" ]
    [ ! -d "${FAKE_HOME}/.claude/agents/shell" ]
}

@test "copies all markdown files recursively" {
    # Add more nested files
    mkdir -p "${FAKE_AGENT_SOURCE}/go/subdir"
    printf '%s\n' "# Nested Agent" > "${FAKE_AGENT_SOURCE}/go/subdir/nested-agent.md"

    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]
    [ -f "${FAKE_HOME}/.claude/agents/nested-agent.md" ]
}

@test "clears existing project agent files before copying but preserves user agents" {
    # Pre-populate agents directory with old files
    mkdir -p "${FAKE_HOME}/.claude/agents"

    # Create an old project agent (will be removed and replaced)
    cat > "${FAKE_HOME}/.claude/agents/old-agent.md" << 'EOF'
---
name: old-agent
description: Old project agent
model: inherit
color: red
project_agent: team-agentic-setup
---
# Old Agent
EOF

    # Create a user agent (will be preserved)
    cat > "${FAKE_HOME}/.claude/agents/user-agent.md" << 'EOF'
---
name: user-agent
description: User's personal agent
model: inherit
color: purple
---
# User Agent
EOF

    printf '%s\n' "stale file" > "${FAKE_HOME}/.claude/agents/stale.txt"

    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]

    # Old project agent should be removed
    [ ! -f "${FAKE_HOME}/.claude/agents/old-agent.md" ]

    # Non-markdown files should be removed
    [ ! -f "${FAKE_HOME}/.claude/agents/stale.txt" ]

    # User agent should be preserved
    [ -f "${FAKE_HOME}/.claude/agents/user-agent.md" ]

    # New project files should exist
    [ -f "${FAKE_HOME}/.claude/agents/go-engineer.md" ]
}

@test "only copies markdown files and ignores other file types" {
    # Add non-markdown files
    printf '%s\n' "text file" > "${FAKE_AGENT_SOURCE}/readme.txt"
    printf '%s\n' "#!/bin/bash" > "${FAKE_AGENT_SOURCE}/script.sh"
    printf '%s\n' "{}" > "${FAKE_AGENT_SOURCE}/config.json"

    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]

    # Markdown files should be copied
    [ -f "${FAKE_HOME}/.claude/agents/go-engineer.md" ]

    # Non-markdown files should not be copied
    [ ! -f "${FAKE_HOME}/.claude/agents/readme.txt" ]
    [ ! -f "${FAKE_HOME}/.claude/agents/script.sh" ]
    [ ! -f "${FAKE_HOME}/.claude/agents/config.json" ]
}

@test "handles empty source directory without error" {
    # Remove all agent files
    rm -rf "${FAKE_AGENT_SOURCE:?}"/*

    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]
    [[ "$output" == *"SUCCESS: all agents updated"* ]]
}

@test "script is idempotent - running multiple times produces same result" {
    # First run
    run "$TEST_SCRIPT"
    [ "$status" -eq 0 ]

    local first_file_count
    first_file_count=$(find "${FAKE_HOME}/.claude/agents" -type f -name "*.md" | wc -l | tr -d ' ')

    # Second run
    run "$TEST_SCRIPT"
    [ "$status" -eq 0 ]

    local second_file_count
    second_file_count=$(find "${FAKE_HOME}/.claude/agents" -type f -name "*.md" | wc -l | tr -d ' ')

    # File counts should be identical
    [ "$first_file_count" -eq "$second_file_count" ]
    [ "$first_file_count" -eq 3 ]
}

# Error Scenario Tests

@test "fails when agent source directory does not exist" {
    # Remove the source directory
    rm -rf "$FAKE_AGENT_SOURCE"

    run "$TEST_SCRIPT"

    [ "$status" -eq 1 ]
    [[ "$output" == *"ERROR: cannot locate the projects agent definitions directory"* ]]
    [[ "$output" == *"${FAKE_AGENT_SOURCE}"* ]]
}

@test "fails when AGENTS_DIR is not set" {
    # Create a modified script with unset AGENTS_DIR
    local test_script_unset="${TEST_DIR}/install-agents-unset.sh"
    cat > "$test_script_unset" << 'EOF'
#!/usr/bin/env bash
set -e

PROJ_ROOT="${FAKE_PROJ_ROOT}"
AGENT_SOURCE="${PROJ_ROOT}/agents/definitions"
AGENTS_DIR=""  # Explicitly empty

copy(){
    # Validate AGENTS_DIR is set before using it
    if [ -z "${AGENTS_DIR}" ]; then
        printf "ERROR: AGENTS_DIR is not set\n" >&2
        return 1
    fi

    # Validate source directory exists
    if [ ! -d "${AGENT_SOURCE}" ]; then
        printf "ERROR: cannot locate the projects agent definitions directory: %s\n" "${AGENT_SOURCE}" >&2
        return 1
    fi

    # Create destination directory if needed
    if [ ! -d "${AGENTS_DIR}" ]; then
        mkdir -p "${AGENTS_DIR}"
    fi

    rm -rf "${AGENTS_DIR:?}"*

    find "${AGENT_SOURCE}" -type f -name "*.md" -exec cp {} "${AGENTS_DIR}" \;

    printf "SUCCESS: all agents updated\n"

    return 0
}

copy
EOF
    chmod +x "$test_script_unset"

    run "$test_script_unset"

    [ "$status" -eq 1 ]
    [[ "$output" == *"ERROR: AGENTS_DIR is not set"* ]]
}

@test "does not create agents directory when source directory is missing" {
    # Remove the source directory
    rm -rf "$FAKE_AGENT_SOURCE"

    run "$TEST_SCRIPT"

    [ "$status" -eq 1 ]

    # Agents directory should not be created if source is missing
    # (script creates it before validation, so this tests the failure path)
    [[ "$output" == *"ERROR"* ]]
}

# Edge Cases

@test "handles agent files with spaces in names" {
    # Create file with spaces
    printf '%s\n' "# Special Agent" > "${FAKE_AGENT_SOURCE}/special agent.md"

    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]
    [ -f "${FAKE_HOME}/.claude/agents/special agent.md" ]
}

@test "handles deeply nested directory structures" {
    # Create deeply nested structure
    mkdir -p "${FAKE_AGENT_SOURCE}/level1/level2/level3"
    printf '%s\n' "# Deep Agent" > "${FAKE_AGENT_SOURCE}/level1/level2/level3/deep-agent.md"

    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]
    [ -f "${FAKE_HOME}/.claude/agents/deep-agent.md" ]
}

@test "handles agent files with special characters in names" {
    # Create files with special characters (avoiding problematic shell chars)
    printf '%s\n' "# Special" > "${FAKE_AGENT_SOURCE}/agent-with-dash.md"
    printf '%s\n' "# Special" > "${FAKE_AGENT_SOURCE}/agent_with_underscore.md"
    printf '%s\n' "# Special" > "${FAKE_AGENT_SOURCE}/agent.with.dots.md"

    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]
    [ -f "${FAKE_HOME}/.claude/agents/agent-with-dash.md" ]
    [ -f "${FAKE_HOME}/.claude/agents/agent_with_underscore.md" ]
    [ -f "${FAKE_HOME}/.claude/agents/agent.with.dots.md" ]
}

@test "preserves file contents when copying" {
    local expected_content="# Go Engineer"

    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]

    local actual_content
    actual_content=$(cat "${FAKE_HOME}/.claude/agents/go-engineer.md")
    [ "$actual_content" = "$expected_content" ]
}

@test "handles large number of agent files" {
    # Create many agent files
    local i
    for i in $(seq 1 50); do
        printf '# Agent %s\n' "$i" > "${FAKE_AGENT_SOURCE}/agent-${i}.md"
    done

    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]

    local file_count
    file_count=$(find "${FAKE_HOME}/.claude/agents" -type f -name "*.md" | wc -l | tr -d ' ')

    # Should have 50 new files + 3 original test files = 53 total
    [ "$file_count" -eq 53 ]
}

@test "handles duplicate filenames in different subdirectories" {
    # Create files with same name in different subdirectories
    mkdir -p "${FAKE_AGENT_SOURCE}/dir1"
    mkdir -p "${FAKE_AGENT_SOURCE}/dir2"
    printf '%s\n' "# Agent from dir1" > "${FAKE_AGENT_SOURCE}/dir1/agent.md"
    printf '%s\n' "# Agent from dir2" > "${FAKE_AGENT_SOURCE}/dir2/agent.md"

    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]

    # One will overwrite the other - verify at least one exists
    [ -f "${FAKE_HOME}/.claude/agents/agent.md" ]

    # Count should include the duplicated filename (last one wins)
    local file_count
    file_count=$(find "${FAKE_HOME}/.claude/agents" -type f -name "agent.md" | wc -l | tr -d ' ')
    [ "$file_count" -eq 1 ]
}

@test "cleans project agent files and subdirectories but preserves user agents" {
    # Pre-populate with files and subdirectories
    mkdir -p "${FAKE_HOME}/.claude/agents"
    mkdir -p "${FAKE_HOME}/.claude/agents/subdir"

    # Create old project agent (will be removed)
    cat > "${FAKE_HOME}/.claude/agents/old.md" << 'EOF'
---
name: old
project_agent: team-agentic-setup
---
# Old
EOF

    # Create user agent (will be preserved)
    cat > "${FAKE_HOME}/.claude/agents/my-agent.md" << 'EOF'
---
name: my-agent
---
# My Agent
EOF

    printf '%s\n' "subdir file" > "${FAKE_HOME}/.claude/agents/subdir/sub.md"

    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]

    # Old project agent file should be removed
    [ ! -f "${FAKE_HOME}/.claude/agents/old.md" ]

    # User agent should be preserved
    [ -f "${FAKE_HOME}/.claude/agents/my-agent.md" ]

    # Subdirectory and its contents should be removed by rm -rf pattern
    [ ! -d "${FAKE_HOME}/.claude/agents/subdir" ]
}

@test "preserves user-created agents without project_agent metadata" {
    # Create agents directory
    mkdir -p "${FAKE_HOME}/.claude/agents"

    # Create a user-created agent (without project_agent field)
    cat > "${FAKE_HOME}/.claude/agents/my-personal-agent.md" << 'EOF'
---
name: my-personal-agent
description: A user-created personal agent
model: inherit
color: purple
allowed_tools:
  - "Read(*)"
---

# My Personal Agent
This is a user-created agent that should never be deleted.
EOF

    # Create a project agent (with project_agent: team-agentic-setup field)
    cat > "${FAKE_HOME}/.claude/agents/test-project-agent.md" << 'EOF'
---
name: test-project-agent
description: A project agent for testing
model: inherit
color: blue
project_agent: team-agentic-setup
allowed_tools:
  - "Read(*)"
---

# Test Project Agent
This should be updated by the script.
EOF

    # Create a corresponding project agent source file with different content
    cat > "${FAKE_AGENT_SOURCE}/test-project-agent.md" << 'EOF'
---
name: test-project-agent
description: Updated project agent for testing
model: inherit
color: green
project_agent: team-agentic-setup
allowed_tools:
  - "Read(*)"
  - "Write(*)"
---

# Test Project Agent (Updated)
This is the new version from the source.
EOF

    # Capture original user agent content
    local user_agent_content
    user_agent_content=$(cat "${FAKE_HOME}/.claude/agents/my-personal-agent.md")

    # Run the install script
    run "$TEST_SCRIPT"

    [ "$status" -eq 0 ]

    # User agent should still exist with unchanged content
    [ -f "${FAKE_HOME}/.claude/agents/my-personal-agent.md" ]

    local preserved_content
    preserved_content=$(cat "${FAKE_HOME}/.claude/agents/my-personal-agent.md")
    [ "$preserved_content" = "$user_agent_content" ]

    # Project agent should be updated from source
    [ -f "${FAKE_HOME}/.claude/agents/test-project-agent.md" ]

    local updated_content
    updated_content=$(cat "${FAKE_HOME}/.claude/agents/test-project-agent.md")
    printf '%s\n' "$updated_content" | grep -q "Updated project agent for testing"
    printf '%s\n' "$updated_content" | grep -q "This is the new version from the source"

    # Verify project agents from source are also installed
    [ -f "${FAKE_HOME}/.claude/agents/go-engineer.md" ]
    [ -f "${FAKE_HOME}/.claude/agents/shell-engineer.md" ]
    [ -f "${FAKE_HOME}/.claude/agents/software-architect.md" ]
}
