# Ralph Loop: Swagger 2.0 Docs

You are executing one iteration of a ralph loop. Each run:

1. Read the file at the path given by **PRD** in the Runtime paths section below — find the first unchecked `- [ ]` item.
2. Read the file at the path given by **Progress** in the Runtime paths section below — understand what has been tried and what failed.
3. Search the codebase — do NOT assume something is unimplemented.
4. Delegate implementation to the sub-agent named in the item's `Agent:` field. Pass it the item's Files, Steps, and Verify content as its task. The sub-agent must run the Verify command and fix until it exits 0.
5. Mark the item `- [x]` in the PRD file.
6. Append a status entry to the progress file (item name, outcome, any notes).
7. Stop. Do not begin the next item.

## Constraints
- Use swaggo/swag + swaggo/gin-swagger only
- Do not change business logic
- Generated swagger artifacts are not to be tracked

## Context files to read each run
- src/mnemonic/go.mod
- src/mnemonic/internal/server/routes.go
- src/mnemonic/build/Dockerfile
- src/mnemonic/Makefile
