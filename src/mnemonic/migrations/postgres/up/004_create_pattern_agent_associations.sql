-- src/migrations/postgres/up/004_create_pattern_agent_associations.sql
-- Creates the pattern-agent association table for many-to-many relationships.
-- Part of Mnemonic MVP
--
-- Dependencies:
--   - 002_create_agents (for agents table)
--   - 003_create_patterns (for patterns table)
--
-- This table establishes which patterns are relevant to which agents,
-- with a relevance score indicating how strongly they relate.
-- Used by the routing engine to find patterns applicable to a selected agent.

create table if not exists pattern_agent_associations (
    -- Composite primary key: one association per pattern-agent pair
    pattern_id uuid not null,
    agent_name varchar(64) not null,

    -- Relevance score from 0.0 (minimally relevant) to 1.0 (highly relevant)
    -- Used for ranking patterns when multiple are associated with an agent
    relevance double precision not null,

    -- Foreign key to patterns table
    -- CASCADE: if pattern is deleted, remove all its associations
    constraint fk_pattern_agent_assoc_pattern
        foreign key (pattern_id) references patterns(id) on delete cascade,

    -- Foreign key to agents table
    -- CASCADE: if agent is deleted, remove all its associations
    constraint fk_pattern_agent_assoc_agent
        foreign key (agent_name) references agents(name) on delete cascade,

    -- Primary key: composite of pattern_id and agent_name
    primary key (pattern_id, agent_name),

    -- Relevance must be in the valid range [0.0, 1.0]
    constraint pattern_agent_assoc_relevance_range
        check (relevance >= 0 and relevance <= 1)
);

-- Indexes for foreign key lookups
-- These improve query performance when:
-- 1. Finding all agents associated with a pattern
-- 2. Finding all patterns associated with an agent

-- Index for lookups by pattern_id (find agents for a pattern)
create index idx_pattern_agent_assoc_pattern
    on pattern_agent_associations(pattern_id);

-- Index for lookups by agent_name (find patterns for an agent)
create index idx_pattern_agent_assoc_agent
    on pattern_agent_associations(agent_name);

-- Table and column documentation
comment on table pattern_agent_associations is
    'Many-to-many relationship between patterns and agents with relevance scores';
comment on column pattern_agent_associations.pattern_id is
    'Reference to the pattern (UUID from patterns table)';
comment on column pattern_agent_associations.agent_name is
    'Reference to the agent (name from agents table)';
comment on column pattern_agent_associations.relevance is
    'Relevance score from 0.0 (not relevant) to 1.0 (highly relevant)';
