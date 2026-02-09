---
name: code-review
description: Orchestrates a 3-agent parallel code review combining pattern compliance, Go conventions, and architectural analysis.
project_skill: team-agentic-setup
allowed-tools:
  - Task
  - Read
  - Glob
  - Grep
  - Bash
---

# Code Review Skill

Orchestrate a comprehensive code review using three specialist agents in parallel, then synthesize their findings.

## Inputs

This Skill reads `$ARGUMENTS`. Accept these patterns:

- A file path, directory path, or glob pattern to review
- A PR number (e.g., `#42` or `42`)
- `--diff` to review staged/unstaged git changes
- No arguments defaults to reviewing uncommitted changes (`git diff`)

If the user didn't supply arguments, ask what they want reviewed.

## Step-by-step procedure

### Step 1: Determine review scope

Identify what code to review:

- **PR**: Run `git diff <base>...<head>` to get changed files
- **Files/directories**: Use the provided paths
- **Staged changes**: Run `git diff --cached`
- **Uncommitted changes**: Run `git diff`

Collect the list of changed/target files and their contents.

### Step 2: Delegate to three agents in parallel

Launch all three agents simultaneously using the Task tool:

1. **`code-review-agent`** (tactical)
   - Pattern compliance, linting, best practices
   - Prompt: Provide the file paths and ask it to review against Cognee patterns, run linters, and return structured findings

2. **`software-architect-agent`** (strategic)
   - Architectural concerns, design coherence
   - Prompt: Provide the file paths and ask it to evaluate architectural consistency, separation of concerns, and design coherence

3. **`go-architect-agent`** (Go-specific)
   - Go coding conventions, naming conventions, implementation conventions
   - Prompt: Provide the Go file paths and ask it to check for idiomatic Go (stuttering, interface design, error wrapping, context propagation, package structure, naming)

### Step 3: Synthesize findings

When all three agents return:

1. **Identify agreements** — Issues flagged by multiple agents
2. **Identify unique findings** — Issues only one agent caught
3. **Identify disagreements** — Conflicting recommendations

### Step 4: Reconcile disagreements

If disagreements exist:

1. Present the conflicting views back to each disagreeing agent
2. Ask each to provide reasoning
3. Request they reach consensus
4. If no consensus, present both perspectives to the user

### Step 5: Compile unified findings

Present to the user in this format:

```markdown
## Code Review: [scope description]

### Summary

[1-2 sentence overall assessment]

### High Priority

- [ ] **[Category]** `file:line` - [Description]
  - Source: [which agent(s) flagged this]
  - Suggested fix: [specific remediation]

### Medium Priority

- [ ] **[Category]** `file:line` - [Description]
  - Suggested fix: [specific remediation]

### Low Priority / Suggestions

- [ ] **[Category]** `file:line` - [Description]

### Good Patterns Observed

- [Well-implemented patterns worth preserving]

### Patterns to Document

- [New patterns that should be added to Cognee]
```

### Step 6: Collaborate with user on resolution

- Discuss trade-offs for architectural decisions
- Get user approval before delegating fixes
- Do NOT auto-fix without user consent

### Step 7: Delegate approved fixes

After user approves specific fixes, delegate to appropriate specialists:

- Go issues → `go-software-agent`
- Shell issues → `shell-script-agent`
- DevOps issues → `go-devops-agent`
- Documentation → `documentation-agent`

## Key principles

- Three perspectives: tactical (code-review) + Go-specific (go-architect) + strategic (software-architect)
- Agents reconcile disagreements BEFORE presenting to user
- User is involved in resolution decisions, not just notified
- Documentation updated to capture learnings from review
- Pattern compliance checked against Cognee knowledge base

## Finding categories

- **Pattern Violation**: Deviates from documented Cognee pattern
- **Go Idiom**: Non-idiomatic Go (stuttering, interface placement, error handling)
- **Architecture**: Structural or design concern
- **Security**: Potential security issue
- **Error Handling**: Missing or inadequate error handling
- **Testing**: Missing tests or edge cases
- **Performance**: Inefficient implementation
- **Style**: Naming, formatting, organization issues
