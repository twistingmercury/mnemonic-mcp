# Code Review: [Phase/Scope Title]

**Review Date:** YYYY-MM-DD
**Reviewers:** [reviewer-1], [reviewer-2], [reviewer-3]
**Phase:** [phase-number] ([phase title])

## Files Reviewed

### Source Files

- `[path/to/file]` - [short description]

### Test Files

- `[path/to/test_file]`

## Validation Results

| Tool   | Result   |
| ------ | -------- |
| [tool] | [result] |

## Design Compliance

Implementation satisfies all [phase/design] behavioral requirements from [design doc path].

### Behavioral Requirements Verified

- [requirement] ✓

### Design Doc Divergences (Post-Review)

Code review fixes introduced divergences between the implementation and the design/architecture docs. A compliance audit identified the following. All divergences are justified improvements — the design docs are being updated to match the implementation.

#### Naming Divergences ([count] renames applied to code, docs updated to match)

| Old Name (in design docs) | New Name (in implementation) | Reason   |
| ------------------------- | ---------------------------- | -------- |
| `[OldName]`               | `[NewName]`                  | [reason] |

#### Structural Divergences (justified improvements over design doc)

| Divergence   | Design Doc   | Implementation   | Assessment   |
| ------------ | ------------ | ---------------- | ------------ |
| [divergence] | [design doc] | [implementation] | [assessment] |

#### Documents Updated

| Document     | Scope   | Status   |
| ------------ | ------- | -------- |
| `[doc path]` | [scope] | [status] |

## Findings

### HIGH Priority

| ID | Source         | Finding   | Resolution   |
| -- | -------------- | --------- | ------------ |
| H1 | [agent/source] | [finding] | [resolution] |

### MEDIUM Priority

| ID | Source         | Finding   | Resolution   |
| -- | -------------- | --------- | ------------ |
| M1 | [agent/source] | [finding] | [resolution] |

### LOW Priority

| ID | Source         | Finding   | Resolution   |
| -- | -------------- | --------- | ------------ |
| L1 | [agent/source] | [finding] | [resolution] |

## Patterns to Document

Patterns identified that should be added to the patterns and examples for Claude Code's sub agents.

1. [pattern]

## Notes for Future Phases

**Phase [n]** ([title]): [note]
