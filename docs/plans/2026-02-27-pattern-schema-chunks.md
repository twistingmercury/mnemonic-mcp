# Pattern Schema & Chunk-Based Search Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Align the `patterns` table with the real pattern file schema (add `entity_type`, `language`, `domain`, `version`, `related_patterns`) and introduce chunk-based semantic search via a new `pattern_chunks` table.

**Architecture:** Pattern files are split into H2-bounded chunks at write time. Each chunk gets its own embedding; search queries `pattern_chunks.embedding` and returns chunk content plus parent pattern metadata. The enrichment pipeline is updated to operate per-chunk, rolling up aggregate status to the parent pattern.

**Tech Stack:** Go 1.25, PostgreSQL + PGVector, Neo4j, golang-migrate, gopkg.in/yaml.v3, Gin, zerolog

**Design doc:** `docs/plans/2026-02-27-pattern-schema-chunks-design.md`

---

## Task 1: DB Migration

**Files:**
- Create: `src/migrations/postgres/000009_pattern_schema_chunks.up.sql`
- Create: `src/migrations/postgres/000009_pattern_schema_chunks.down.sql`

**Step 1: Write the up migration**

```sql
-- 000009_pattern_schema_chunks.up.sql

-- Drop the 10KB content constraint
alter table patterns drop constraint if exists patterns_content_length;

-- Drop embedding from patterns (moves to pattern_chunks)
alter table patterns drop column if exists embedding;

-- Add structured metadata columns
alter table patterns
    add column entity_type varchar(100) not null default '',
    add column language    varchar(50)  not null default '',
    add column domain      varchar(50)  not null default '',
    add column version     varchar(50),
    add column related_patterns jsonb not null default '[]'::jsonb;

alter table patterns
    add constraint patterns_related_patterns_array
        check (jsonb_typeof(related_patterns) = 'array');

-- Index for common filters
create index on patterns (language);
create index on patterns (domain);
create index on patterns (entity_type);

-- New pattern_chunks table
create table if not exists pattern_chunks (
    id                uuid    primary key default gen_random_uuid(),
    pattern_id        uuid    not null references patterns(id) on delete cascade,
    section_title     varchar(255) not null,
    chunk_index       int     not null,
    content           text    not null,
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
create index on pattern_chunks using ivfflat (embedding vector_cosine_ops) with (lists = 100);

-- Add chunk_id to enrichment_jobs; make pattern_id nullable
alter table enrichment_jobs
    add column chunk_id uuid references pattern_chunks(id) on delete cascade;
alter table enrichment_jobs
    alter column pattern_id drop not null;

comment on table pattern_chunks is 'H2-bounded chunks of patterns with per-chunk embeddings for semantic search';
comment on column pattern_chunks.chunk_index is 'Zero-based order within parent pattern';
```

**Step 2: Write the down migration**

```sql
-- 000009_pattern_schema_chunks.down.sql

alter table enrichment_jobs
    alter column pattern_id set not null;
alter table enrichment_jobs
    drop column if exists chunk_id;

drop table if exists pattern_chunks;

drop index if exists patterns_language_idx;
drop index if exists patterns_domain_idx;
drop index if exists patterns_entity_type_idx;

alter table patterns
    drop constraint if exists patterns_related_patterns_array,
    drop column if exists entity_type,
    drop column if exists language,
    drop column if exists domain,
    drop column if exists version,
    drop column if exists related_patterns;

alter table patterns
    add column embedding vector(1536);

alter table patterns
    add constraint patterns_content_length
        check (length(content) <= 10240);
```

**Step 3: Verify migration runs cleanly**

```bash
cd src/mnemonic/tests
docker compose -f docker-compose-dev.yaml up -d postgres migrate
docker logs tests_migrate_1 --follow
# Expected: migrate exits 0, no errors
docker compose -f docker-compose-dev.yaml down -v
```

**Step 4: Commit**

```bash
git add src/migrations/postgres/000009_pattern_schema_chunks.up.sql \
        src/migrations/postgres/000009_pattern_schema_chunks.down.sql
git commit -m "feat: add pattern_chunks table and structured metadata columns"
```

---

## Task 2: Chunk Repository Package

**Files:**
- Create: `src/mnemonic/internal/repository/chunk/doc.go`
- Create: `src/mnemonic/internal/repository/chunk/errors.go`
- Create: `src/mnemonic/internal/repository/chunk/chunk.go`
- Create: `src/mnemonic/internal/repository/chunk/repository.go`
- Create: `src/mnemonic/internal/repository/chunk/repository_test.go`

**Step 1: Write doc.go**

```go
// Package chunk provides the repository layer for pattern chunks.
// Each chunk represents one H2-bounded section of a parent pattern,
// with its own vector embedding for semantic similarity search.
package chunk
```

**Step 2: Write errors.go**

```go
package chunk

import "errors"

// ErrNotFound is returned when a chunk does not exist.
var ErrNotFound = errors.New("chunk not found")
```

**Step 3: Write chunk.go**

```go
package chunk

import (
    "time"
    "github.com/google/uuid"
)

// Chunk is one H2-bounded section of a parent pattern.
type Chunk struct {
    ID               uuid.UUID  `db:"id"`
    PatternID        uuid.UUID  `db:"pattern_id"`
    SectionTitle     string     `db:"section_title"`
    ChunkIndex       int        `db:"chunk_index"`
    Content          string     `db:"content"`
    Embedding        []float32  `db:"-"`
    EnrichmentStatus string     `db:"enrichment_status"`
    EnrichmentError  *string    `db:"enrichment_error"`
    EnrichedAt       *time.Time `db:"enriched_at"`
    CreatedAt        time.Time  `db:"created_at"`
    UpdatedAt        time.Time  `db:"updated_at"`
}

// Match is a similarity search result from pattern_chunks joined to patterns.
type Match struct {
    ChunkID      uuid.UUID
    PatternID    uuid.UUID
    PatternName  string
    EntityType   string
    Language     string
    Domain       string
    Tags         []string
    SectionTitle string
    ChunkIndex   int
    Content      string
    Similarity   float64
}
```

**Step 4: Write the failing test for Repository**

```go
// repository_test.go
package chunk_test

import (
    "context"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    chunkrepo "github.com/twistingmercury/mnemonic/internal/repository/chunk"
)

func TestRepository_Create_Get_Delete(t *testing.T) {
    repo := chunkrepo.NewRepository(testDB)
    patternID := uuid.New() // assume pattern exists via test setup

    chunk := &chunkrepo.Chunk{
        PatternID:    patternID,
        SectionTitle: "Overview",
        ChunkIndex:   0,
        Content:      "This is the overview section.",
    }

    err := repo.Create(context.Background(), chunk)
    require.NoError(t, err)
    assert.NotEqual(t, uuid.Nil, chunk.ID)

    got, err := repo.Get(context.Background(), chunk.ID)
    require.NoError(t, err)
    assert.Equal(t, "Overview", got.SectionTitle)

    err = repo.DeleteByPatternID(context.Background(), patternID)
    require.NoError(t, err)
}

func TestRepository_UpdateEmbedding(t *testing.T) {
    // ... test UpdateEmbedding and UpdateEnrichmentStatus
}

func TestRepository_FindSimilar(t *testing.T) {
    // ... test similarity search returns ChunkMatch with parent pattern fields
}
```

**Step 5: Write repository.go**

```go
package chunk

import (
    "context"
    "database/sql"
    "encoding/json"
    "errors"
    "fmt"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/pgvector/pgvector-go"
    "github.com/twistingmercury/mnemonic/internal/repository"
)

// Repository defines the data access interface for pattern chunks.
type Repository interface {
    // Create inserts a new chunk. Sets ID, CreatedAt, UpdatedAt on the chunk.
    Create(ctx context.Context, chunk *Chunk) error

    // CreateBatch inserts multiple chunks for a pattern in one transaction.
    CreateBatch(ctx context.Context, chunks []*Chunk) error

    // Get retrieves a chunk by ID.
    Get(ctx context.Context, id uuid.UUID) (*Chunk, error)

    // ListByPatternID returns all chunks for a pattern, ordered by chunk_index.
    ListByPatternID(ctx context.Context, patternID uuid.UUID) ([]*Chunk, error)

    // DeleteByPatternID removes all chunks for a pattern.
    DeleteByPatternID(ctx context.Context, patternID uuid.UUID) error

    // UpdateEmbedding stores the embedding for a chunk.
    UpdateEmbedding(ctx context.Context, id uuid.UUID, embedding []float32) error

    // UpdateEnrichmentStatus updates the enrichment state for a chunk.
    UpdateEnrichmentStatus(ctx context.Context, id uuid.UUID, status string, errMsg *string) error

    // FindSimilar performs pgvector cosine similarity search against chunk embeddings.
    // Joins to patterns to return parent metadata alongside chunk content.
    FindSimilar(ctx context.Context, embedding []float32, opts SimilarityOptions) ([]*Match, error)

    // AllEnrichedForPattern returns true if every chunk for patternID has status 'enriched'.
    AllEnrichedForPattern(ctx context.Context, patternID uuid.UUID) (bool, error)

    // AnyFailedForPattern returns true if any chunk for patternID has status 'failed'.
    AnyFailedForPattern(ctx context.Context, patternID uuid.UUID) (bool, error)
}

// SimilarityOptions controls similarity search behaviour.
type SimilarityOptions struct {
    MinSimilarity float64
    MaxResults    int
    Tags          []string  // filter by parent pattern tags
    Language      string    // filter by parent pattern language
    Domain        string    // filter by parent pattern domain
    PatternIDs    []uuid.UUID
}

type pgRepository struct {
    pool *pgxpool.Pool
}

// NewRepository creates a Repository backed by the given connection pool.
func NewRepository(pool *pgxpool.Pool) Repository {
    return &pgRepository{pool: pool}
}

func (r *pgRepository) Create(ctx context.Context, c *Chunk) error {
    const q = `
        insert into pattern_chunks
            (pattern_id, section_title, chunk_index, content, updated_at)
        values ($1, $2, $3, $4, now())
        returning id, enrichment_status, created_at, updated_at`

    return r.pool.QueryRow(ctx, q,
        c.PatternID, c.SectionTitle, c.ChunkIndex, c.Content,
    ).Scan(&c.ID, &c.EnrichmentStatus, &c.CreatedAt, &c.UpdatedAt)
}

func (r *pgRepository) CreateBatch(ctx context.Context, chunks []*Chunk) error {
    tx, err := r.pool.Begin(ctx)
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer tx.Rollback(ctx)

    for _, c := range chunks {
        const q = `
            insert into pattern_chunks
                (pattern_id, section_title, chunk_index, content, updated_at)
            values ($1, $2, $3, $4, now())
            returning id, enrichment_status, created_at, updated_at`
        if err := tx.QueryRow(ctx, q,
            c.PatternID, c.SectionTitle, c.ChunkIndex, c.Content,
        ).Scan(&c.ID, &c.EnrichmentStatus, &c.CreatedAt, &c.UpdatedAt); err != nil {
            return fmt.Errorf("insert chunk %d: %w", c.ChunkIndex, err)
        }
    }
    return tx.Commit(ctx)
}

func (r *pgRepository) Get(ctx context.Context, id uuid.UUID) (*Chunk, error) {
    const q = `
        select id, pattern_id, section_title, chunk_index, content,
               enrichment_status, enrichment_error, enriched_at, created_at, updated_at
        from pattern_chunks where id = $1`

    c := &Chunk{}
    err := r.pool.QueryRow(ctx, q, id).Scan(
        &c.ID, &c.PatternID, &c.SectionTitle, &c.ChunkIndex, &c.Content,
        &c.EnrichmentStatus, &c.EnrichmentError, &c.EnrichedAt, &c.CreatedAt, &c.UpdatedAt,
    )
    if errors.Is(err, sql.ErrNoRows) {
        return nil, ErrNotFound
    }
    return c, err
}

func (r *pgRepository) ListByPatternID(ctx context.Context, patternID uuid.UUID) ([]*Chunk, error) {
    const q = `
        select id, pattern_id, section_title, chunk_index, content,
               enrichment_status, enrichment_error, enriched_at, created_at, updated_at
        from pattern_chunks where pattern_id = $1 order by chunk_index`

    rows, err := r.pool.Query(ctx, q, patternID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var chunks []*Chunk
    for rows.Next() {
        c := &Chunk{}
        if err := rows.Scan(
            &c.ID, &c.PatternID, &c.SectionTitle, &c.ChunkIndex, &c.Content,
            &c.EnrichmentStatus, &c.EnrichmentError, &c.EnrichedAt, &c.CreatedAt, &c.UpdatedAt,
        ); err != nil {
            return nil, err
        }
        chunks = append(chunks, c)
    }
    return chunks, rows.Err()
}

func (r *pgRepository) DeleteByPatternID(ctx context.Context, patternID uuid.UUID) error {
    _, err := r.pool.Exec(ctx,
        `delete from pattern_chunks where pattern_id = $1`, patternID)
    return err
}

func (r *pgRepository) UpdateEmbedding(ctx context.Context, id uuid.UUID, embedding []float32) error {
    _, err := r.pool.Exec(ctx,
        `update pattern_chunks set embedding = $1, updated_at = now() where id = $2`,
        pgvector.NewVector(embedding), id)
    return err
}

func (r *pgRepository) UpdateEnrichmentStatus(ctx context.Context, id uuid.UUID, status string, errMsg *string) error {
    var enrichedAt *time.Time
    if status == "enriched" {
        now := time.Now()
        enrichedAt = &now
    }
    _, err := r.pool.Exec(ctx, `
        update pattern_chunks
        set enrichment_status = $1,
            enrichment_error  = $2,
            enriched_at       = $3,
            updated_at        = now()
        where id = $4`,
        status, errMsg, enrichedAt, id)
    return err
}

func (r *pgRepository) FindSimilar(ctx context.Context, embedding []float32, opts SimilarityOptions) ([]*Match, error) {
    const q = `
        select
            pc.id, pc.pattern_id, pc.section_title, pc.chunk_index, pc.content,
            p.name, p.entity_type, p.language, p.domain, p.tags,
            1 - (pc.embedding <=> $1) as similarity
        from pattern_chunks pc
        join patterns p on p.id = pc.pattern_id
        where pc.embedding is not null
          and 1 - (pc.embedding <=> $1) >= $2
          and ($3::text = '' or p.language = $3)
          and ($4::text = '' or p.domain   = $4)
        order by pc.embedding <=> $1
        limit $5`

    rows, err := r.pool.Query(ctx, q,
        pgvector.NewVector(embedding),
        opts.MinSimilarity,
        opts.Language,
        opts.Domain,
        opts.MaxResults,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var matches []*Match
    for rows.Next() {
        m := &Match{}
        var tagsRaw []byte
        if err := rows.Scan(
            &m.ChunkID, &m.PatternID, &m.SectionTitle, &m.ChunkIndex, &m.Content,
            &m.PatternName, &m.EntityType, &m.Language, &m.Domain, &tagsRaw,
            &m.Similarity,
        ); err != nil {
            return nil, err
        }
        _ = json.Unmarshal(tagsRaw, &m.Tags)
        matches = append(matches, m)
    }
    return matches, rows.Err()
}

func (r *pgRepository) AllEnrichedForPattern(ctx context.Context, patternID uuid.UUID) (bool, error) {
    var count int
    err := r.pool.QueryRow(ctx, `
        select count(*) from pattern_chunks
        where pattern_id = $1 and enrichment_status != 'enriched'`, patternID,
    ).Scan(&count)
    return count == 0, err
}

func (r *pgRepository) AnyFailedForPattern(ctx context.Context, patternID uuid.UUID) (bool, error) {
    var count int
    err := r.pool.QueryRow(ctx, `
        select count(*) from pattern_chunks
        where pattern_id = $1 and enrichment_status = 'failed'`, patternID,
    ).Scan(&count)
    return count > 0, err
}

// compile-time check
var _ Repository = (*pgRepository)(nil)
```

**Step 6: Run tests**

```bash
cd src/mnemonic
go test ./internal/repository/chunk/... -v
```

**Step 7: Commit**

```bash
git add src/mnemonic/internal/repository/chunk/
git commit -m "feat: add chunk repository package"
```

---

## Task 3: Update Pattern Repository

**Files:**
- Modify: `src/mnemonic/internal/repository/pattern/pattern.go`
- Modify: `src/mnemonic/internal/repository/pattern/repository.go`
- Modify: `src/mnemonic/internal/repository/pattern/repository_test.go`

**Step 1: Update Pattern struct** in `pattern.go`

Add these fields (remove `Embedding []float32`):

```go
// EntityType is the pattern category (e.g., "go-pattern", "e2e-testing").
EntityType string `db:"entity_type"`

// Language is the programming language (e.g., "go", "agnostic", "shell").
Language string `db:"language"`

// Domain is the technical domain (e.g., "backend", "api-design", "testing").
Domain string `db:"domain"`

// Version is the optional target version (e.g., "Go 1.21+").
Version *string `db:"version"`

// RelatedPatterns is a list of related pattern entity names.
// Stored as JSONB.
RelatedPatterns []string `db:"-"`
```

Also update `Filter`:

```go
type Filter struct {
    Tags             []string
    EnrichmentStatus string
    SearchQuery      string
    Language         string  // new
    Domain           string  // new
    EntityType       string  // new
}
```

**Step 2: Update SQL in repository.go**

- `Create`: add `entity_type`, `language`, `domain`, `version`, `related_patterns` to INSERT
- `Update`: add new fields to SET clause
- `Get`: add new fields to SELECT and Scan
- `List`: add new filter conditions for `language`, `domain`, `entity_type`
- Remove `UpdateEmbedding` method (embeddings now live on chunks)
- Keep `UpdateEnrichmentStatus` — it still sets aggregate status on the parent

**Step 3: Run tests**

```bash
cd src/mnemonic
go test ./internal/repository/pattern/... -v
```

Expected: existing tests pass with updated struct fields.

**Step 4: Commit**

```bash
git add src/mnemonic/internal/repository/pattern/
git commit -m "feat: add entity_type, language, domain, version, related_patterns to pattern repo"
```

---

## Task 4: Update EnrichmentJob Type and Repository

**Files:**
- Modify: `src/mnemonic/internal/repository/enrichmentjob/enrichmentjob.go`
- Modify: `src/mnemonic/internal/repository/enrichmentjob/repository.go`
- Modify: `src/mnemonic/internal/repository/enrichmentjob/repository_test.go`

**Step 1: Update Job struct** — add `ChunkID`, make `PatternID` a pointer

```go
// PatternID is set for legacy jobs. Nil for chunk-based jobs.
PatternID *uuid.UUID `db:"pattern_id"`

// ChunkID is set for chunk-based enrichment jobs.
ChunkID *uuid.UUID `db:"chunk_id"`
```

Update `Filter`:

```go
type Filter struct {
    Status    *string
    PatternID *uuid.UUID
    ChunkID   *uuid.UUID  // new
}
```

**Step 2: Update Create in repository.go**

The `Create` method should accept either `PatternID` or `ChunkID` being set:

```go
func (r *pgRepository) Create(ctx context.Context, job *Job) error {
    const q = `
        insert into enrichment_jobs
            (pattern_id, chunk_id, status, max_attempts, scheduled_for, updated_at)
        values ($1, $2, $3, $4, now(), now())
        returning id, attempts, created_at, updated_at`
    return r.pool.QueryRow(ctx, q,
        job.PatternID, job.ChunkID,
        string(StatusPending), DefaultMaxAttempts,
    ).Scan(&job.ID, &job.Attempts, &job.CreatedAt, &job.UpdatedAt)
}
```

**Step 3: Run tests**

```bash
cd src/mnemonic
go test ./internal/repository/enrichmentjob/... -v
```

**Step 4: Commit**

```bash
git add src/mnemonic/internal/repository/enrichmentjob/
git commit -m "feat: add chunk_id to enrichment job, make pattern_id nullable"
```

---

## Task 5: Update Pattern Service — New Fields + Chunking

**Files:**
- Modify: `src/mnemonic/internal/service/pattern/service.go`
- Modify: `src/mnemonic/internal/service/pattern/service_test.go`

**Step 1: Write failing test for chunk splitting**

```go
func TestSplitChunks_ByH2Heading(t *testing.T) {
    content := `## Philosophy
Storage-only databases.

## created_at Handling
Let the database set it.

## updated_at Handling
Always set explicitly.`

    chunks := splitIntoChunks(content)
    require.Len(t, chunks, 3)
    assert.Equal(t, "Philosophy", chunks[0].SectionTitle)
    assert.Equal(t, 0, chunks[0].ChunkIndex)
    assert.Contains(t, chunks[0].Content, "Storage-only")
    assert.Equal(t, "updated_at Handling", chunks[2].SectionTitle)
    assert.Equal(t, 2, chunks[2].ChunkIndex)
}

func TestSplitChunks_NoH2_SingleChunk(t *testing.T) {
    content := "Just a paragraph with no headings."
    chunks := splitIntoChunks(content)
    require.Len(t, chunks, 1)
    assert.Equal(t, "Content", chunks[0].SectionTitle)
    assert.Equal(t, content, chunks[0].Content)
}

func TestSplitChunks_ContentBeforeFirstH2(t *testing.T) {
    content := `Preamble text here.

## First Section
Section content.`

    chunks := splitIntoChunks(content)
    require.Len(t, chunks, 2)
    assert.Equal(t, "Overview", chunks[0].SectionTitle)
    assert.Contains(t, chunks[0].Content, "Preamble")
}
```

**Step 2: Run test to verify it fails**

```bash
cd src/mnemonic
go test ./internal/service/pattern/... -run TestSplitChunks -v
# Expected: FAIL — splitIntoChunks not defined
```

**Step 3: Implement splitIntoChunks**

Add to `service.go` (unexported helper):

```go
// splitChunk is a parsed chunk from content.
type splitChunk struct {
    SectionTitle string
    ChunkIndex   int
    Content      string
}

// splitIntoChunks splits markdown content at H2 boundaries.
// Content before the first H2 becomes an "Overview" chunk.
// Content with no H2 headings becomes a single "Content" chunk.
func splitIntoChunks(content string) []splitChunk {
    lines := strings.Split(content, "\n")
    var chunks []splitChunk
    var currentTitle string
    var currentLines []string
    index := 0

    flush := func(title string) {
        body := strings.TrimSpace(strings.Join(currentLines, "\n"))
        if body == "" {
            return
        }
        chunks = append(chunks, splitChunk{
            SectionTitle: title,
            ChunkIndex:   index,
            Content:      body,
        })
        index++
    }

    foundH2 := false
    for _, line := range lines {
        if strings.HasPrefix(line, "## ") {
            if !foundH2 {
                flush("Overview")
                foundH2 = true
            } else {
                flush(currentTitle)
            }
            currentTitle = strings.TrimPrefix(line, "## ")
            currentLines = nil
        } else {
            currentLines = append(currentLines, line)
        }
    }
    if foundH2 {
        flush(currentTitle)
    } else {
        flush("Content")
    }
    return chunks
}
```

**Step 4: Update CreateInput and UpdateInput**

```go
type CreateInput struct {
    Name              string
    EntityType        string
    Language          string
    Domain            string
    Version           *string
    Description       *string
    Content           string
    Tags              []string
    RelatedPatterns   []string
    AgentAssociations []AssociationInput
}
```

(Same additions to `UpdateInput`.)

**Step 5: Update Create to chunk content and queue per-chunk jobs**

`Create` now:
1. Resolves agent associations (best-effort)
2. Creates pattern record with new fields
3. Calls `splitIntoChunks(input.Content)`
4. Calls `chunkRepo.CreateBatch` to insert chunks
5. Creates one `enrichmentjob.Job{ChunkID: &chunk.ID}` per chunk

The `patternService` struct gains `chunkRepo chunkrepo.Repository`.

**Step 6: Run tests**

```bash
cd src/mnemonic
go test ./internal/service/pattern/... -v
```

**Step 7: Commit**

```bash
git add src/mnemonic/internal/service/pattern/
git commit -m "feat: add chunking to pattern service, update CreateInput/UpdateInput"
```

---

## Task 6: Update Enrichment Service — Chunk-Based Pipeline

**Files:**
- Modify: `src/mnemonic/internal/service/enrichment/service.go`
- Modify: `src/mnemonic/internal/service/enrichment/service_test.go`

**Step 1: Update ProcessJob**

New pipeline for chunk-based jobs (`job.ChunkID != nil`):

```
1. Load chunk from chunkRepo by job.ChunkID
2. Generate embedding for chunk.Content
3. Store embedding in pattern_chunks via chunkRepo.UpdateEmbedding
4. Mark chunk enriched via chunkRepo.UpdateEnrichmentStatus
5. Check aggregate: if chunkRepo.AnyFailedForPattern → pattern 'failed'
              else if chunkRepo.AllEnrichedForPattern → run concept extraction, mark pattern 'enriched'
6. Mark job completed
```

Concept extraction and Neo4j sync run **once** when the aggregate transitions to `enriched`:

```go
if allDone {
    pattern, _ := s.patternRepo.Get(ctx, chunk.PatternID)
    concepts, _ := s.extractionSvc.Extract(ctx, pattern.Content)
    s.syncConceptsAndGraph(ctx, pattern, concepts)
    s.patternRepo.UpdateEnrichmentStatus(ctx, pattern.ID, "enriched", nil)
}
```

Legacy jobs (`job.ChunkID == nil`, `job.PatternID != nil`) keep the old code path so existing data isn't broken.

**Step 2: Update failJob**

When failing a chunk job, also update the parent pattern status:

```go
func (s *enrichmentService) failChunkJob(ctx context.Context, job *enrichmentjob.Job, chunkID uuid.UUID, patternID uuid.UUID, cause error) error {
    errMsg := cause.Error()
    s.chunkRepo.UpdateEnrichmentStatus(ctx, chunkID, "failed", &errMsg)
    s.patternRepo.UpdateEnrichmentStatus(ctx, patternID, "failed", &errMsg)
    return s.jobRepo.MarkFailed(ctx, job.ID, cause, s.cfg.RetryDelay)
}
```

**Step 3: Run tests**

```bash
cd src/mnemonic
go test ./internal/service/enrichment/... -v
```

**Step 4: Commit**

```bash
git add src/mnemonic/internal/service/enrichment/
git commit -m "feat: update enrichment service to operate on chunks"
```

---

## Task 7: Update Search Service — Query Chunk Embeddings

**Files:**
- Modify: `src/mnemonic/internal/service/search/service.go`
- Modify: `src/mnemonic/internal/service/search/service_test.go`

**Step 1: Update SearchResult type**

```go
// ChunkMatch is a single semantic search hit from a pattern chunk.
type ChunkMatch struct {
    PatternID    uuid.UUID
    PatternName  string
    EntityType   string
    Language     string
    Domain       string
    Tags         []string
    SectionTitle string
    ChunkIndex   int
    Content      string
    Similarity   float64
}

type SearchResult struct {
    Matches          []*ChunkMatch
    Query            string
    TotalCandidates  int
    SearchDurationMs int64
}
```

**Step 2: Update SearchPatterns to use chunkRepo.FindSimilar**

Replace `patternRepo.FindSimilar` with `chunkRepo.FindSimilar`. Map `chunk.Match` → `ChunkMatch`.

Add `Language` and `Domain` to `SearchOptions` (optional filters).

**Step 3: Run tests**

```bash
cd src/mnemonic
go test ./internal/service/search/... -v
```

**Step 4: Commit**

```bash
git add src/mnemonic/internal/service/search/
git commit -m "feat: update search service to query chunk embeddings"
```

---

## Task 8: Update Pattern Handler

**Files:**
- Modify: `src/mnemonic/internal/handlers/patterns/patterns.go`
- Modify: `src/mnemonic/internal/handlers/patterns/patterns_test.go`

**Step 1: Update request/response types**

Add to `patternCreateRequest` / `patternUpdateRequest`:

```go
EntityType      string   `json:"entity_type"`
Language        string   `json:"language"`
Domain          string   `json:"domain"`
Version         *string  `json:"version"`
RelatedPatterns []string `json:"related_patterns"`
```

Add to `patternResponse`:

```go
EntityType      string          `json:"entity_type"`
Language        string          `json:"language"`
Domain          string          `json:"domain"`
Version         *string         `json:"version"`
RelatedPatterns []string        `json:"related_patterns"`
Chunks          []chunkSummary  `json:"chunks,omitempty"`
```

Where:

```go
type chunkSummary struct {
    ChunkIndex       int    `json:"chunk_index"`
    SectionTitle     string `json:"section_title"`
    EnrichmentStatus string `json:"enrichment_status"`
}
```

**Step 2: Update validation**

Add to `validatePatternFields`:

- `entity_type`: required, max 100 chars, kebab-case (`^[a-z][a-z0-9-]*$`)
- `language`: required, one of allowed values from schema
- `domain`: required, one of allowed values from schema

**Step 3: Register new chunks endpoint**

```go
rg.GET("/patterns/:id/chunks", h.GetChunks)
```

`GetChunks` loads chunks via `chunkRepo.ListByPatternID` and returns full content.

**Step 4: Update List handler**

Add `language`, `domain`, `entity_type` query params to filter.

**Step 5: Update Search response**

`searchResultResponse` now maps from `search.ChunkMatch`:

```go
type searchResultResponse struct {
    PatternID    string   `json:"pattern_id"`
    PatternName  string   `json:"pattern_name"`
    EntityType   string   `json:"entity_type"`
    Language     string   `json:"language"`
    Domain       string   `json:"domain"`
    Tags         []string `json:"tags"`
    SectionTitle string   `json:"section_title"`
    ChunkIndex   int      `json:"chunk_index"`
    Content      string   `json:"content"`
    Similarity   float64  `json:"similarity"`
}
```

**Step 6: Run tests**

```bash
cd src/mnemonic
go test ./internal/handlers/patterns/... -v
```

**Step 7: Commit**

```bash
git add src/mnemonic/internal/handlers/patterns/
git commit -m "feat: update pattern handler for new metadata fields and chunk endpoint"
```

---

## Task 9: Wire Up New Dependencies

**Files:**
- Modify: `src/mnemonic/internal/mcpserver/deps.go`
- Modify: `src/mnemonic/internal/server/routes.go`
- Modify: `src/mnemonic/cmd/main/main.go`

**Step 1: Add chunkRepo to deps**

In `deps.go`, instantiate `chunkrepo.NewRepository(pool)` and pass it to pattern service, enrichment service, search service, and pattern handler.

**Step 2: Register chunks route**

In `routes.go`, ensure `GET /v1/api/patterns/:id/chunks` is registered via the pattern handler.

**Step 3: Build to verify wiring**

```bash
cd src/mnemonic
go build ./...
# Expected: no errors
```

**Step 4: Commit**

```bash
git add src/mnemonic/internal/mcpserver/deps.go \
        src/mnemonic/internal/server/routes.go \
        src/mnemonic/cmd/main/main.go
git commit -m "chore: wire chunk repository into services and handlers"
```

---

## Task 10: Pattern Loader Binary

**Files:**
- Create: `src/mnemonic/cmd/loader/main.go`

**Step 1: Write the loader**

```go
// cmd/loader/main.go loads pattern files from a directory into Mnemonic via the Admin API.
//
// Usage:
//   loader --dir /path/to/patterns --api-url http://localhost:8080
package main

import (
    "bytes"
    "encoding/json"
    "flag"
    "fmt"
    "io/fs"
    "net/http"
    "os"
    "path/filepath"
    "strings"

    "gopkg.in/yaml.v3"
)

type frontmatter struct {
    EntityName     string   `yaml:"entity_name"`
    EntityType     string   `yaml:"entity_type"`
    Language       string   `yaml:"language"`
    Domain         string   `yaml:"domain"`
    Description    string   `yaml:"description"`
    Tags           []string `yaml:"tags"`
    Version        string   `yaml:"version"`
    RelatedPatterns []string `yaml:"related_patterns"`
}

type patternRequest struct {
    Name            string   `json:"name"`
    EntityType      string   `json:"entity_type"`
    Language        string   `json:"language"`
    Domain          string   `json:"domain"`
    Version         *string  `json:"version,omitempty"`
    Description     string   `json:"description,omitempty"`
    Content         string   `json:"content"`
    Tags            []string `json:"tags"`
    RelatedPatterns []string `json:"related_patterns"`
}

func main() {
    dir := flag.String("dir", "", "directory containing pattern .md files (required)")
    apiURL := flag.String("api-url", "http://localhost:8080", "Mnemonic Admin API base URL")
    flag.Parse()

    if *dir == "" {
        fmt.Fprintln(os.Stderr, "error: --dir is required")
        os.Exit(1)
    }

    var failed int
    err := filepath.WalkDir(*dir, func(path string, d fs.DirEntry, err error) error {
        if err != nil || d.IsDir() {
            return err
        }
        if !strings.HasSuffix(path, ".md") {
            return nil
        }
        base := filepath.Base(path)
        if base == "README.md" || base == "PATTERN-METADATA-SCHEMA.md" {
            return nil
        }
        if loadErr := loadFile(path, *apiURL); loadErr != nil {
            fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", path, loadErr)
            failed++
        } else {
            fmt.Printf("OK   %s\n", path)
        }
        return nil
    })
    if err != nil {
        fmt.Fprintf(os.Stderr, "walk error: %v\n", err)
        os.Exit(1)
    }
    if failed > 0 {
        fmt.Fprintf(os.Stderr, "\n%d file(s) failed\n", failed)
        os.Exit(1)
    }
}

func loadFile(path, apiURL string) error {
    raw, err := os.ReadFile(path)
    if err != nil {
        return err
    }

    fm, content, err := parseFrontmatter(string(raw))
    if err != nil {
        return fmt.Errorf("parse frontmatter: %w", err)
    }

    name := slugFromFilename(filepath.Base(path))

    req := patternRequest{
        Name:            name,
        EntityType:      fm.EntityType,
        Language:        fm.Language,
        Domain:          fm.Domain,
        Description:     fm.Description,
        Content:         content,
        Tags:            fm.Tags,
        RelatedPatterns: fm.RelatedPatterns,
    }
    if fm.Version != "" {
        req.Version = &fm.Version
    }
    if req.Tags == nil {
        req.Tags = []string{}
    }
    if req.RelatedPatterns == nil {
        req.RelatedPatterns = []string{}
    }

    body, _ := json.Marshal(req)

    // Try POST first; fall back to PUT on 409.
    resp, err := http.Post(apiURL+"/v1/api/patterns", "application/json", bytes.NewReader(body))
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusConflict {
        putReq, _ := http.NewRequest(http.MethodPut,
            apiURL+"/v1/api/patterns/"+name, bytes.NewReader(body))
        putReq.Header.Set("Content-Type", "application/json")
        putResp, putErr := http.DefaultClient.Do(putReq)
        if putErr != nil {
            return putErr
        }
        defer putResp.Body.Close()
        if putResp.StatusCode >= 300 {
            return fmt.Errorf("PUT returned %d", putResp.StatusCode)
        }
        return nil
    }

    if resp.StatusCode >= 300 {
        return fmt.Errorf("POST returned %d", resp.StatusCode)
    }
    return nil
}

func parseFrontmatter(raw string) (*frontmatter, string, error) {
    parts := strings.SplitN(raw, "---", 3)
    if len(parts) < 3 {
        return nil, raw, fmt.Errorf("no YAML frontmatter found")
    }
    var fm frontmatter
    if err := yaml.Unmarshal([]byte(parts[1]), &fm); err != nil {
        return nil, "", err
    }
    return &fm, strings.TrimSpace(parts[2]), nil
}

func slugFromFilename(name string) string {
    return strings.TrimSuffix(name, ".md")
}
```

**Step 2: Build the loader**

```bash
cd src/mnemonic
go build ./cmd/loader/...
# Expected: produces ./loader binary (or cmd/loader/loader)
```

**Step 3: Commit**

```bash
git add src/mnemonic/cmd/loader/
git commit -m "feat: add pattern loader binary"
```

---

## Task 11: Update E2E Tests

**Files:**
- Modify: `src/mnemonic/tests/e2e/types.go`
- Modify: `src/mnemonic/tests/e2e/patterns_test.go`

**Step 1: Update types.go**

Add to `Pattern`, `PatternCreate`, `PatternUpdate`:

```go
EntityType      string   `json:"entity_type"`
Language        string   `json:"language"`
Domain          string   `json:"domain"`
Version         string   `json:"version,omitempty"`
RelatedPatterns []string `json:"related_patterns,omitempty"`
```

Add new types:

```go
type ChunkSummary struct {
    ChunkIndex       int    `json:"chunk_index"`
    SectionTitle     string `json:"section_title"`
    EnrichmentStatus string `json:"enrichment_status"`
}

type ChunkDetail struct {
    ChunkIndex       int    `json:"chunk_index"`
    SectionTitle     string `json:"section_title"`
    Content          string `json:"content"`
    EnrichmentStatus string `json:"enrichment_status"`
}

type ChunkList struct {
    Data []ChunkDetail `json:"data"`
}
```

Update `PatternSearchResult` to match new chunk-based response fields.

**Step 2: Update patterns_test.go**

- Update helper functions that create patterns to include `entity_type`, `language`, `domain`
- Add `TestGetPatternChunks_ReturnsChunksForPattern`
- Add `TestGetPatternChunks_NotFound`
- Update search tests to assert on new `section_title`, `pattern_name` fields in results

**Step 3: Run E2E tests**

```bash
cd src/mnemonic/tests
bash run-e2e.sh
# Expected: all tests pass
```

**Step 4: Commit**

```bash
git add src/mnemonic/tests/e2e/
git commit -m "test: update e2e tests for pattern chunks and new metadata fields"
```

---

## Task 12: Update Architecture Docs

Dispatch the `technical-writer` agent to:

1. Update `docs/architecture/04-data-architecture.md` — add `pattern_chunks` table, updated `patterns` columns, updated `enrichment_jobs`
2. Update `docs/design/pattern-processing.md` — describe chunk-based enrichment pipeline
3. Update `docs/design/service-layer.md` — updated `PatternService`, `EnrichmentService`, `SearchService` interfaces
4. Update `docs/design/mcp-server.md` — updated search result shape (chunk content + parent metadata)

---

## Running the Loader After Implementation

With the API running:

```bash
cd src/mnemonic
./loader --dir /Users/doublej/dev/claudecode/patterns --api-url http://localhost:8080
```

Expected output: one `OK` line per pattern file, exit 0.
