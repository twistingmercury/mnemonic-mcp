# Pattern Schema & Chunk-Based Search Design

**Status:** Approved
**Date:** 2026-02-27
**Context:** Pattern files at `/Users/doublej/dev/claudecode/patterns/PATTERN-METADATA-SCHEMA.md`

## Problem

The current `patterns` table does not store metadata fields present in every pattern file's YAML frontmatter (`entity_type`, `language`, `domain`, `version`, `related_patterns`). Pattern files also range from 5KB to 18KB, exceeding the 10KB `CHECK` constraint. Embedding a whole file as a single vector dilutes semantic search quality — an agent asking about `updated_at` handling should not have to compete with sqlx examples and anti-patterns in the same embedding.

## Approach

Store full pattern metadata in `patterns`. Chunk each pattern's content by H2 section into a new `pattern_chunks` table. Embed at the chunk level. Search against chunk embeddings, return chunk content plus parent metadata.

## Data Model

### `patterns` table changes

- Drop `patterns_content_length` CHECK constraint
- Drop `embedding vector(1536)` — embeddings move to chunks
- Keep `enrichment_status`, `enrichment_error`, `enriched_at` as aggregate status over all chunks
- Add columns:

| Column             | Type                          | Notes                                   |
| ------------------ | ----------------------------- | --------------------------------------- |
| `entity_type`      | `varchar(100) not null`       | e.g. `go-pattern`, `e2e-testing`        |
| `language`         | `varchar(50) not null`        | e.g. `go`, `agnostic`, `shell`, `sql`   |
| `domain`           | `varchar(50) not null`        | e.g. `backend`, `api-design`, `testing` |
| `version`          | `varchar(50)`                 | nullable, e.g. `Go 1.21+`               |
| `related_patterns` | `jsonb not null default '[]'` | array of entity name strings            |

### `pattern_chunks` table (new)

```sql
create table pattern_chunks (
    id                uuid primary key default gen_random_uuid(),
    pattern_id        uuid not null references patterns(id) on delete cascade,
    section_title     varchar(255) not null,
    chunk_index       int not null,
    content           text not null,
    embedding         vector(1536),
    enrichment_status varchar(20) not null default 'pending',
    enrichment_error  text,
    enriched_at       timestamptz,
    created_at        timestamptz not null default now(),
    updated_at        timestamptz not null default now(),
    constraint pattern_chunks_enrichment_status_valid
        check (enrichment_status in ('pending', 'enriched', 'failed'))
);

create index on pattern_chunks (pattern_id);
create index on pattern_chunks using ivfflat (embedding vector_cosine_ops);
```

### `enrichment_jobs` table changes

- Add `chunk_id uuid references pattern_chunks(id) on delete cascade`
- Make `pattern_id` nullable — existing rows retained, new jobs use `chunk_id`

## API Changes

### Pattern write (POST/PUT)

Request gains `entity_type`, `language`, `domain`, `version`, `related_patterns`. On write, the handler:

1. Creates/updates the `patterns` row
2. Parses content into H2-bounded chunks
3. Replaces all `pattern_chunks` rows for the pattern
4. Queues one enrichment job per chunk

### Pattern read (GET `/v1/api/patterns/:id`)

Response includes new metadata fields plus a `chunks` array:

```json
{
  "entity_type": "go-pattern",
  "language": "go",
  "domain": "backend",
  "version": "Go 1.21+",
  "related_patterns": ["SQL Migration Pattern"],
  "chunks": [
    {
      "chunk_index": 0,
      "section_title": "Philosophy",
      "enrichment_status": "enriched"
    },
    {
      "chunk_index": 1,
      "section_title": "created_at Handling",
      "enrichment_status": "enriched"
    }
  ]
}
```

No chunk content inline — use the chunks endpoint for that.

### Pattern list (GET `/v1/api/patterns`)

Gains `language`, `domain`, `entity_type` filter query parameters.

### Pattern search (GET `/v1/api/patterns/search`)

Interface unchanged (`q`, `limit`, `threshold`, `tags`). Internally queries `pattern_chunks.embedding` via cosine similarity. Results return:

- Matching chunk's `content` and `section_title`
- Parent pattern metadata (`name`, `entity_type`, `language`, `domain`, `tags`)
- `similarity` score

### New: chunk content (GET `/v1/api/patterns/:id/chunks`)

Returns full chunk content for a pattern. For tooling and debugging.

## Enrichment Pipeline

- Enricher picks up pending jobs by `chunk_id`
- Embeds chunk content (not full pattern content)
- On completion, updates `pattern_chunks.enrichment_status`
- After all chunks for a pattern complete, rolls up to `patterns.enrichment_status`

**Aggregate status rules:**

| Condition           | `patterns.enrichment_status` |
| ------------------- | ---------------------------- |
| No chunks enriched  | `pending`                    |
| All chunks enriched | `enriched`                   |
| Any chunk failed    | `failed`                     |

**Neo4j / concept extraction:** runs once per pattern against full content. Graph relationships stay at the pattern level.

## Pattern Loader

Standalone Go binary at `cmd/loader/main.go`.

**Flags:**

- `--dir` — path to pattern directory
- `--api-url` — Mnemonic Admin API base URL (default `http://localhost:8080`)

**Behavior:**

1. Walk all `.md` files; skip `README.md` and `PATTERN-METADATA-SCHEMA.md`
2. Parse YAML frontmatter per file
3. Derive `name` slug from filename (already kebab-case)
4. POST to `/v1/api/patterns`; on 409 conflict, PUT to update
5. Report per-file result; exit non-zero if any file failed

**Out of scope for MVP:** dry-run mode, delete-removed patterns, parallel loading.

## Out of Scope

- MCP tool changes (`search_patterns` returns chunks already — no interface change needed)
- Authentication
- Chunk-level Neo4j nodes
