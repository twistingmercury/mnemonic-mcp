# Phase 15: Migration Cleanup

> Part of the [MVP Implementation Plan](mvp-implementation-plan.md)

**Goal:** Remove the routing_rules migration and trim the performance indexes migration to eliminate routing-related database artifacts.

**Agent(s):** data-engineer

**Dependencies:** Phase 14 (routing engine complete)

---

Since there are no deployed databases, we delete the files and edit in place.

## Step 1: Delete migration 005 (up)

- Delete file: `/Users/doublej/dev/mnemonic/src/migrations/postgres/up/005_create_routing_rules.sql`
- Agent: `data-engineer`
- Verify: `ls /Users/doublej/dev/mnemonic/src/migrations/postgres/up/005_create_routing_rules.sql` returns "No such file"

## Step 2: Delete migration 005 (down)

- Delete file: `/Users/doublej/dev/mnemonic/src/migrations/postgres/down/005_create_routing_rules.sql`
- Agent: `data-engineer`
- Verify: `ls /Users/doublej/dev/mnemonic/src/migrations/postgres/down/005_create_routing_rules.sql` returns "No such file"

## Step 3: Trim migration 007 (up) -- remove routing indexes

- Modify file: `/Users/doublej/dev/mnemonic/src/migrations/postgres/up/007_create_performance_indexes.sql`
- Remove the entire `ROUTING RULES INDEXES` section (lines creating `idx_routing_rules_enabled_priority` and its comment)
- Remove the `005_create_routing_rules` dependency from the file header comment
- Keep the `PATTERNS INDEXES` section and `ENRICHMENT JOBS INDEXES` section intact
- Agent: `data-engineer`
- Verify: `grep -c routing_rules /Users/doublej/dev/mnemonic/src/migrations/postgres/up/007_create_performance_indexes.sql` returns `0`

## Step 4: Trim migration 007 (down) -- remove routing index drop

- Modify file: `/Users/doublej/dev/mnemonic/src/migrations/postgres/down/007_create_performance_indexes.sql`
- Remove the line `drop index if exists idx_routing_rules_enabled_priority;`
- Remove the `-- Drop routing rules indexes` comment
- Keep the enrichment jobs and patterns index drops intact
- Agent: `data-engineer`
- Verify: `grep -c routing_rules /Users/doublej/dev/mnemonic/src/migrations/postgres/down/007_create_performance_indexes.sql` returns `0`

## Step 5: Verify remaining migrations are consistent

- Run: `ls -1 /Users/doublej/dev/mnemonic/src/migrations/postgres/up/` -- should show 001, 002, 003, 004, 006, 007 (no 005)
- Run: `ls -1 /Users/doublej/dev/mnemonic/src/migrations/postgres/down/` -- should show 001, 002, 003, 004, 006, 007 (no 005)
- Agent: `data-engineer`

## Step 6: Commit

```bash
git add src/migrations/postgres/up/ src/migrations/postgres/down/
git commit -m "feat(pivot): delete routing_rules migration, trim routing indexes from 007"
```
