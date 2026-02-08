-- src/mnemonic/migrations/postgres/up/006_create_enrichment_jobs.sql
-- Creates the enrichment_jobs table for background pattern processing.
-- Part of Mnemonic MVP
--
-- Dependencies: 003_create_patterns (for patterns table)
--
-- Enrichment jobs track the background processing of patterns:
-- - Embedding generation via OpenAI text-embedding-3-small
-- - Entity extraction for Neo4j knowledge graph
-- - Pattern-agent association calculation
--
-- This table serves as a simple Postgres-backed job queue using
-- FOR UPDATE SKIP LOCKED for safe concurrent processing across
-- multiple Mnemonic pods.
--
-- Job lifecycle:
--   pending -> processing -> completed (success)
--                        -> failed (error, may retry)
--
-- Note: updated_at is managed by the application layer (Go repository)
-- rather than database triggers for better control and testability.

create table if not exists enrichment_jobs (
    -- UUID primary key
    id uuid primary key default gen_random_uuid(),

    -- Reference to pattern being enriched
    -- CASCADE: if pattern is deleted, remove orphan jobs automatically
    pattern_id uuid not null,

    -- Job processing state
    -- pending: awaiting processing
    -- processing: currently being processed by a worker
    -- completed: successfully finished
    -- failed: processing failed (see last_error)
    status varchar(20) not null default 'pending',

    -- Retry tracking
    -- attempts: number of times processing has been attempted
    -- max_attempts: maximum retries before giving up (default 3)
    attempts integer not null default 0,
    max_attempts integer not null default 3,

    -- Error information
    -- Stores the last error message if processing failed
    last_error text,

    -- Scheduling and timing
    -- scheduled_for: when the job should be processed (supports delayed retry)
    -- started_at: when processing began (null if pending)
    -- completed_at: when processing finished (null if not completed)
    scheduled_for timestamptz not null default now(),
    started_at timestamptz,
    completed_at timestamptz,

    -- Audit timestamps
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),

    -- Foreign key with cascade delete
    -- When a pattern is deleted, all its enrichment jobs are automatically removed
    constraint fk_enrichment_jobs_pattern
        foreign key (pattern_id) references patterns(id) on delete cascade,

    -- Constraints

    -- Status must be one of the valid states
    constraint enrichment_jobs_status_valid
        check (status in ('pending', 'processing', 'completed', 'failed')),

    -- Attempts must be non-negative
    constraint enrichment_jobs_attempts_valid
        check (attempts >= 0),

    -- Max attempts must be at least 1
    constraint enrichment_jobs_max_attempts_valid
        check (max_attempts >= 1)
);

-- Index for pattern lookups (find jobs by pattern)
create index if not exists idx_enrichment_jobs_pattern
    on enrichment_jobs(pattern_id);

-- Table and column documentation
comment on table enrichment_jobs is 'Background processing queue for pattern enrichment';
comment on column enrichment_jobs.id is 'Unique job identifier';
comment on column enrichment_jobs.pattern_id is 'Pattern being enriched (FK to patterns table)';
comment on column enrichment_jobs.status is 'Job state: pending, processing, completed, or failed';
comment on column enrichment_jobs.attempts is 'Number of processing attempts made';
comment on column enrichment_jobs.max_attempts is 'Maximum attempts before marking permanently failed';
comment on column enrichment_jobs.last_error is 'Error message from most recent failed attempt';
comment on column enrichment_jobs.scheduled_for is 'When the job should be processed (supports delayed retry)';
comment on column enrichment_jobs.started_at is 'Timestamp when processing began';
comment on column enrichment_jobs.completed_at is 'Timestamp when processing completed successfully';
comment on column enrichment_jobs.created_at is 'Timestamp when the job was created';
comment on column enrichment_jobs.updated_at is 'Timestamp when the job was last modified';
