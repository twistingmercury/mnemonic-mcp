-- src/migrations/postgres/up/003_create_patterns.sql
-- Creates the patterns table with PGVector embedding support.
-- Part of Mnemonic MVP
--
-- Dependencies: 001_extensions_and_functions (for vector extension)
--
-- Patterns are reusable context fragments that can be matched to prompts
-- via semantic similarity search. Each pattern has:
-- - Unique name for human reference
-- - Content text (up to 10KB) that provides context
-- - Vector embedding (1536 dimensions for text-embedding-3-small)
-- - Enrichment status tracking for async embedding generation
--
-- Note: updated_at is managed by the application layer (Go repository)
-- rather than database triggers for better control and testability.

create table if not exists patterns (
    -- UUID primary key for stable references (patterns may be renamed)
    id uuid primary key default gen_random_uuid(),

    -- Pattern metadata
    -- Unique name for human reference (e.g., "go-error-handling", "api-design-principles")
    name varchar(128) not null,

    -- Optional description explaining when/how to use this pattern
    description varchar(500),

    -- Pattern content (up to 10KB)
    -- This is the actual context text that will be injected into prompts
    content text not null,

    -- Categorization tags (JSON array)
    -- Example: ["golang", "best-practices", "error-handling"]
    tags jsonb not null default '[]'::jsonb,

    -- Vector embedding for semantic search
    -- 1536 dimensions for OpenAI text-embedding-3-small model
    -- Null until enrichment completes
    embedding vector(1536),

    -- Enrichment processing state
    -- pending: awaiting embedding generation
    -- enriched: embedding generated successfully
    -- failed: embedding generation failed (see enrichment_error)
    enrichment_status varchar(20) not null default 'pending',

    -- Error message if enrichment failed (null if pending or enriched)
    enrichment_error text,

    -- Timestamp when enrichment completed (null if pending or failed)
    enriched_at timestamptz,

    -- Audit timestamps
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),

    -- Constraints

    -- Pattern names must be unique for human reference
    constraint patterns_name_unique unique (name),

    -- Content has a maximum length of 10KB (10240 bytes)
    constraint patterns_content_length
        check (length(content) <= 10240),

    -- tags must be a JSON array
    constraint patterns_tags_array
        check (jsonb_typeof(tags) = 'array'),

    -- enrichment_status must be one of the valid states
    constraint patterns_enrichment_status_valid
        check (enrichment_status in ('pending', 'enriched', 'failed'))
);

-- Table and column documentation
comment on table patterns is 'Reusable context patterns for prompt enrichment';
comment on column patterns.id is 'Stable UUID identifier (patterns may be renamed)';
comment on column patterns.name is 'Unique human-readable name (e.g., go-error-handling)';
comment on column patterns.description is 'Optional description of when/how to use this pattern';
comment on column patterns.content is 'Pattern content text for prompt injection (up to 10KB)';
comment on column patterns.tags is 'JSON array of categorization tags';
comment on column patterns.embedding is 'Vector embedding (1536d) for semantic similarity search';
comment on column patterns.enrichment_status is 'Processing state: pending, enriched, or failed';
comment on column patterns.enrichment_error is 'Error message if enrichment failed';
comment on column patterns.enriched_at is 'Timestamp when enrichment completed successfully';
comment on column patterns.created_at is 'Timestamp when the pattern was created';
comment on column patterns.updated_at is 'Timestamp when the pattern was last modified';

-- Performance indexes

-- Index for vector similarity search using IVFFlat
-- Lists parameter tuned for expected dataset size (adjust as data grows)
-- Note: For optimal performance, REINDEX after loading significant data
create index if not exists idx_patterns_embedding_cosine
    on patterns using ivfflat (embedding vector_cosine_ops)
    with (lists = 100);

-- Index for filtering by enrichment status
-- Used by List() and FindSimilar() queries
create index if not exists idx_patterns_enrichment_status
    on patterns(enrichment_status);

-- Index for full-text search on name and description
-- Used by List() with SearchQuery filter
create index if not exists idx_patterns_search
    on patterns using gin (
        to_tsvector('english', name || ' ' || coalesce(description, ''))
    );
