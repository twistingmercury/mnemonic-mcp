# Ralph Loop: Remove REST API and Migrations

You are executing one iteration of a ralph loop. Read this file at the start of every run, complete exactly one PRD cycle, then stop.

## Objective

Complete exactly one unchecked cycle from the PRD. Verify it. Commit it. Stop.

## Inputs

| Item         | Path                                                                                                      |
| ------------ | --------------------------------------------------------------------------------------------------------- |
| PRD          | `docs/plans/phase-02/part-01/PRD.md`                                                                      |
| Progress log | `ralph/progress.txt`                                                                                      |
| Key context  | `docker-compose.yaml`, `src/mnemonic/internal/server/server.go`, `src/mnemonic/internal/server/routes.go` |

## Non-Negotiable Rules

1. Complete exactly **one** cycle per run. Do not begin a second.
2. Read the PRD first — find the first `- [ ]` item. That is the only item you work on.
3. Read the progress log to understand what has already been attempted.
4. Search the codebase before assuming anything is absent or already done.
5. Delegate implementation to the agent named in the cycle's `Agent:` field. Pass it the Files, Steps, and Verify content.
6. The sub-agent must run the Verify command and fix until it exits 0. Do not mark a cycle done until Verify exits 0.
7. After any Go source change (production or test), the sub-agent must run `cd src/mnemonic && make analyze` and fix any issues it raises.
8. Mark the cycle `- [x]` in the PRD after Verify exits 0.
9. Append one entry to `ralph/progress.txt` (cycle name, outcome, any notable fixes).
10. Stage all modified and deleted files and commit with a short, descriptive message. Do **not** push.

## Repo-Specific Build and Test Rules

- Go module root: `src/mnemonic/`
- Build: `cd src/mnemonic && go build ./...`
- Test: `cd src/mnemonic && go test ./...`
- Lint/vet/security: `cd src/mnemonic && make analyze`
- Full CI build (Docker image): `cd src/mnemonic && make build` — this is the final definition of done for the loop
- Signed commits required: always use `git commit -S`. If signing fails, stop and report.
- Never use `--no-verify` or `--no-gpg-sign`.
- `git rm` is preferred over `rm` for tracked file deletions — it stages the removal automatically.

## Ralph Loop Procedure

**Step 1 — Read the PRD.** Find the first `- [ ]` cycle. That is your only target.

**Step 2 — Read the progress log.** Understand what has been tried and what notes were left.

**Step 3 — Search the codebase.** Confirm the current state matches what the cycle assumes. Do not skip this.

**Step 4 — Read the cycle's Files.** Open each listed file before delegating so the sub-agent has context.

**Step 5 — Delegate to the sub-agent.** Pass the cycle's Agent, Files, Steps, and Verify content. Add these baseline checks to every sub-agent task except Cycle 12 (which uses `make build` as its sole gate):

- After any Go file change: `cd src/mnemonic && go build ./...` must exit 0.
- After any Go file change: `cd src/mnemonic && go vet ./...` must exit 0.
- After any Go file change: `cd src/mnemonic && make analyze` must exit 0 — fix all issues it raises before continuing.
- The cycle's Verify command must exit 0 before the sub-agent returns.

**Step 6 — Mark the PRD.** Change `- [ ]` to `- [x]` for the completed cycle.

**Step 7 — Update the progress log.** Append to `ralph/progress.txt`:

```
[Cycle N — <title>] DONE
- <one-line summary of what was changed>
- Verify: <command> exits 0
- Note: <any surprises or workarounds, or "none">
```

**Step 8 — Commit.** Stage all changed and deleted files. Commit with `git commit -S -m "<short description>"`. Stop. Do not begin the next cycle.

## Failure Modes to Avoid

- Do not mark a cycle done before Verify exits 0.
- Do not skip the progress log read — a prior run may have left a partial state.
- Do not `rm` tracked files — use `git rm` so the deletion is staged.
- Do not push to remote.
- Do not combine two cycles into one commit.
- Do not invent file paths — search the repo if unsure where something lives.

## Output Contract

After completing a cycle you must output:

1. Which cycle was completed (name and number).
2. What files were changed or deleted.
3. The Verify command and its exit code.
4. The commit SHA.
