#!/usr/bin/env bats
#
# ralph.bats — Black-box tests for ralph.sh
#
# Requires: bats-core
# Run:  bats .claude/ralph/tests/ralph.bats

# ---------------------------------------------------------------------------
# Setup / teardown
# ---------------------------------------------------------------------------

setup() {
    # Isolated temp directory for each test.
    TEST_DIR="$(mktemp -d)"
    RALPH_DIR="${TEST_DIR}"
    export RALPH_DIR

    PRD_FILE="${RALPH_DIR}/PRD.md"
    PROMPT_FILE="${RALPH_DIR}/PROMPT.md"

    # Create a minimal PROMPT.md so validation passes by default.
    printf 'Do the thing.\n' > "${PROMPT_FILE}"

    # Stub bin directory prepended to PATH so tests never call the real claude-code.
    STUB_BIN="${TEST_DIR}/bin"
    mkdir -p "${STUB_BIN}"
    export PATH="${STUB_BIN}:${PATH}"

    # Absolute path to the script under test.
    SCRIPT="/Users/doublej/dev/mnemonic/.claude/ralph/ralph.sh"

    # Counter file the stub uses to track how many times it has been called.
    STUB_CALL_COUNT_FILE="${TEST_DIR}/stub_call_count"
    printf '0\n' > "${STUB_CALL_COUNT_FILE}"
    export STUB_CALL_COUNT_FILE
}

teardown() {
    rm -rf "${TEST_DIR}"
}

# ---------------------------------------------------------------------------
# Helper: write a stub claude-code that performs a PRD mutation on a specific
# call number.
#
# Arguments:
#   $1  call number on which to mark the first `- [ ]` as `- [x]`
#       (use 0 to never mark anything)
# ---------------------------------------------------------------------------
write_stub() {
    local mark_on_call="${1:-1}"
    local prd="${RALPH_DIR}/PRD.md"

    cat > "${STUB_BIN}/claude-code" <<STUB
#!/usr/bin/env bash
count=\$(cat "${STUB_CALL_COUNT_FILE}")
count=\$(( count + 1 ))
printf '%s\n' "\${count}" > "${STUB_CALL_COUNT_FILE}"

if [ "${mark_on_call}" -gt 0 ] && [ "\${count}" -eq "${mark_on_call}" ]; then
    perl -i -0pe 's/^- \\[ \\]/- [x]/m' "${prd}"
fi
STUB
    chmod +x "${STUB_BIN}/claude-code"
}

# Helper: return the number of times the stub was invoked.
stub_call_count() {
    cat "${STUB_CALL_COUNT_FILE}"
}

# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------

# Happy path: single item, marked [x] on the very first claude-code call.
@test "happy path: item marked [x] on first attempt — exits 0" {
    printf '%s\n' '- [ ] **Item A**' > "${PRD_FILE}"
    write_stub 1

    run bash "${SCRIPT}"

    [ "${status}" -eq 0 ]
    [[ "${output}" == *"Item completed."* ]]
    [[ "${output}" == *"All items complete."* ]]
    [ "$(stub_call_count)" -eq 1 ]
    grep -q '^- \[x\]' "${PRD_FILE}"
}

# Multi-attempt: item stays [ ] for 2 calls, then gets marked [x] on call 3.
@test "multi-attempt: item stays open for 2 attempts, completes on 3rd — exits 0" {
    printf '%s\n' '- [ ] **Item B**' > "${PRD_FILE}"
    write_stub 3

    MAX_ATTEMPTS=10 run bash "${SCRIPT}"

    [ "${status}" -eq 0 ]
    [[ "${output}" == *"All items complete."* ]]
    [ "$(stub_call_count)" -eq 3 ]
    grep -q '^- \[x\]' "${PRD_FILE}"
}

# Limit exceeded: item never gets marked; after MAX_ATTEMPTS calls it becomes [~].
@test "limit exceeded: item abandoned after MAX_ATTEMPTS, loop continues to next item" {
    # Two items: first will never be completed, second is already done.
    { printf '%s\n' '- [ ] **Item C**'; printf '%s\n' '- [x] **Item D**'; } > "${PRD_FILE}"
    # stub never marks anything
    write_stub 0

    MAX_ATTEMPTS=3 run bash "${SCRIPT}"

    [ "${status}" -eq 0 ]
    [[ "${output}" == *"Attempt limit reached."* ]]
    [[ "${output}" == *"All items complete."* ]]
    # Item C must be abandoned
    grep -q '^- \[~\]' "${PRD_FILE}"
    # The stub was called exactly MAX_ATTEMPTS times (3)
    [ "$(stub_call_count)" -eq 3 ]
}

# All complete: no [ ] items at all — exits 0 immediately without calling stub.
@test "all complete: no open items — exits 0 without calling claude-code" {
    printf '%s\n' '- [x] **Item E**' > "${PRD_FILE}"
    write_stub 0

    run bash "${SCRIPT}"

    [ "${status}" -eq 0 ]
    [[ "${output}" == *"All items complete."* ]]
    [ "$(stub_call_count)" -eq 0 ]
}

# Missing PRD.md → exits 1 with error message.
@test "missing PRD.md — exits 1" {
    # Do not create PRD.md
    write_stub 0

    run bash "${SCRIPT}"

    [ "${status}" -eq 1 ]
    [[ "${output}" == *"PRD.md not found"* ]]
}

# Missing PROMPT.md → exits 1 with error message.
@test "missing PROMPT.md — exits 1" {
    printf '%s\n' '- [ ] **Item F**' > "${PRD_FILE}"
    rm -f "${PROMPT_FILE}"
    write_stub 0

    run bash "${SCRIPT}"

    [ "${status}" -eq 1 ]
    [[ "${output}" == *"PROMPT.md not found"* ]]
}
