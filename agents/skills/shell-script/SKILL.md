---
name: shell-script
description: Orchestrates shell script creation with automatic BATS test generation and iterative fix loop.
project_skill: team-agentic-setup
allowed-tools:
  - Task
  - Read
  - Glob
  - Grep
  - Bash
---

# Shell Script Creation Skill

Orchestrate production-grade shell script creation with automatic BATS test generation and an iterative quality feedback loop.

## Inputs

This Skill reads `$ARGUMENTS`. Accept these patterns:

- Script requirements or purpose (e.g., "Create a backup script for Docker volumes")
- Path where the script should be created
- Environment variables or configuration the script needs
- Any specific requirements (error handling, cross-platform support, etc.)

If the user didn't supply sufficient details, ask for:
- What the script should do
- Where to create it
- Required environment variables
- Expected behavior and error scenarios

## Step-by-step procedure

### Step 1: Gather requirements

Identify what needs to be created:

- **Script purpose**: What the script should accomplish
- **Target path**: Where to create the script file
- **Requirements**: Environment variables, configuration, dependencies
- **Behavior**: Expected output, error scenarios, edge cases

If requirements are incomplete, ask the user for clarification before proceeding.

### Step 2: Delegate to shell-script-agent

Launch `shell-script-agent` using the Task tool:

**Prompt:**
```
Create a shell script at <target-path> that <requirements>.

Requirements:
- <list all requirements>
- Environment variables: <list env vars>
- Error scenarios: <list error cases>
- Cross-platform support: <macOS/Linux if applicable>

Follow all shell scripting standards:
- POSIX compliance
- Variable naming conventions
- Never-nester pattern
- Environment variable configuration
- Shellcheck compliance
```

**Wait for the agent to complete** and return the created script.

### Step 3: AUTOMATICALLY delegate to bats-test-agent

**CRITICAL: Do NOT wait for user request. Testing happens automatically.**

Immediately after the script is created, launch `bats-test-agent` using the Task tool:

**Prompt:**
```
Create BATS tests for the shell script at <script-path>.

The script does: <brief description>

Test scenarios to cover:
- Happy path: <successful execution scenarios>
- Error scenarios: <all error cases from requirements>
- Edge cases: <special inputs, concurrent execution, etc.>

Requirements:
- Test isolation with $BATS_TEST_TMPDIR
- Docker testing if script uses Docker
- Validate output, exit codes, and side effects
- All tests must pass before completion
```

**Wait for the agent to complete** test creation and execution.

### Step 4: Handle test results

The `bats-test-agent` will run the tests and report results.

**If all tests pass:**
- Proceed to Step 7 (report completion)

**If tests fail:**
- Proceed to Step 5 (diagnose and fix)

### Step 5: Diagnose test failures

The `bats-test-agent` will determine the root cause of failures:

- **Test has a bug**: Incorrect expectations, wrong assertions, faulty test logic
- **Script has a bug**: The script doesn't behave as expected

### Step 6: Delegate fixes based on root cause

**If the test has a bug:**
- The `bats-test-agent` will fix the test itself
- Wait for the agent to fix and re-run tests
- Return to Step 4 (handle test results)

**If the script has a bug:**
- The `bats-test-agent` will delegate back to `shell-script-agent` with:
  - Clear description of the bug
  - Current incorrect behavior
  - Expected correct behavior
  - Failing test reference
  - Script location and line numbers
- Wait for `shell-script-agent` to fix the script
- The `bats-test-agent` will automatically re-run tests
- Return to Step 4 (handle test results)

**Iterate Steps 4-6 until all tests pass.**

### Step 7: Report completion to user

Present the final result:

```markdown
## Shell Script Creation Complete

**Script Created**: `<script-path>`

**Test Suite**: `<test-path>`

**Test Results**:
✅ All <N> tests passing

### Usage

Required environment variables:
- `<VAR_NAME>`: <description>
- `<VAR_NAME>`: <description>

Example invocation:
```bash
export VAR_NAME="value"
export VAR_NAME="value"
./<script-name>.sh
```

### Test Coverage

- Happy path scenarios: <N> tests
- Error scenarios: <N> tests
- Edge cases: <N> tests

Run tests with:
```bash
bats <test-file>.bats
```
```

## Key principles

- **Testing happens AUTOMATICALLY** — Don't wait for user to ask
- **The workflow feels seamless** — Script + tests = one operation
- **Quality is ensured** — Test feedback loop catches bugs before delivery
- **Separation of concerns** — Shell script agent creates scripts, BATS test agent creates and runs tests
- **Iterative refinement** — Fix loop continues until all tests pass
- **Never mix responsibilities** — Each agent does only its specialized work

## Workflow guarantees

1. **Scripts are always tested** — No script is delivered without tests
2. **Tests always pass** — Fix loop ensures quality before completion
3. **Bugs are caught early** — Test failures trigger immediate fixes
4. **Clear responsibility** — Test bugs fixed by test agent, script bugs fixed by script agent
5. **User gets working code** — Both script and tests validated before delivery

## Agent coordination

This skill coordinates two specialist agents:

1. **`shell-script-agent`** (script implementation)
   - Creates production-grade shell scripts
   - Applies POSIX compliance, never-nester pattern, SOLID principles
   - Fixes script bugs when reported by test agent

2. **`bats-test-agent`** (test implementation)
   - Creates comprehensive BATS test suites
   - Runs tests and diagnoses failures
   - Fixes test bugs
   - Delegates script bugs back to shell-script-agent

**The skill orchestrates these agents** to deliver validated, tested shell scripts.

## Error handling

- If `shell-script-agent` fails to create script → Report error to user, ask for clarification
- If `bats-test-agent` fails to create tests → Report error to user, review script requirements
- If tests repeatedly fail after fixes → Report to user, provide diagnostic information
- If agents cannot reach consensus on bug cause → Escalate to user for guidance

## Success criteria

The skill completes successfully when:

- ✅ Shell script created following all standards
- ✅ BATS test suite created with comprehensive coverage
- ✅ All tests pass
- ✅ Shellcheck passes with no errors
- ✅ Usage documentation provided
- ✅ Example invocation included
