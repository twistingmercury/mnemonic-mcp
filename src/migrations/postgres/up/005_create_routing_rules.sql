-- src/migrations/postgres/up/005_create_routing_rules.sql
-- Creates the routing_rules table for prompt-to-agent matching.
-- Part of Mnemonic MVP
--
-- Dependencies: 002_create_agents (for agents table)
--
-- Routing rules define how user prompts are matched to specific agents.
-- Rules are evaluated in priority order (highest first), and the first
-- matching rule determines the target agent.
--
-- Match types:
--   - keyword: matches if prompt contains specified keywords
--   - regex: matches if prompt matches a regular expression
--   - pattern: matches via semantic similarity to referenced patterns
--   - default: fallback rule that always matches (lowest priority)
--
-- Note: updated_at is managed by the application layer (Go repository)
-- rather than database triggers for better control and testability.

create table if not exists routing_rules (
    -- UUID primary key (rules may be renamed, need stable reference)
    id uuid primary key default gen_random_uuid(),

    -- Rule metadata
    -- Unique name for human reference (e.g., "go-code-rule", "python-fallback")
    name varchar(128) not null,

    -- Priority for evaluation order (0-1000, higher values evaluated first)
    -- Rules with same priority are ordered by id for deterministic behavior
    priority integer not null,

    -- Target agent for this rule (references agents.name)
    -- RESTRICT: prevent agent deletion if referenced by rules
    agent_name varchar(64) not null,

    -- Match type determines how match_config is interpreted
    match_type varchar(20) not null,

    -- Type-specific match configuration (JSONB)
    -- Structure depends on match_type:
    --   keyword: {"keywords": ["go", "golang"], "match_mode": "any"|"all"}
    --   regex: {"pattern": "\\b(go|golang)\\b", "flags": "i"}
    --   pattern: {"pattern_ids": ["uuid1", "uuid2"]}
    --   default: {} (empty object)
    match_config jsonb not null,

    -- Rule enabled/disabled state
    -- Disabled rules are not evaluated during routing
    enabled boolean not null default true,

    -- Audit timestamps
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),

    -- Foreign key to agents table
    -- RESTRICT prevents deleting an agent that is referenced by routing rules
    constraint fk_routing_rules_agent
        foreign key (agent_name) references agents(name) on delete restrict,

    -- Constraints

    -- Rule names must be unique for human reference
    constraint routing_rules_name_unique unique (name),

    -- Priority must be within valid range
    constraint routing_rules_priority_range
        check (priority >= 0 and priority <= 1000),

    -- Match type must be one of the valid types
    constraint routing_rules_match_type_valid
        check (match_type in ('keyword', 'regex', 'pattern', 'default')),

    -- Match config validation based on match_type
    -- Ensures required keys are present for each match type
    constraint routing_rules_match_config_valid check (
        (match_type = 'keyword' and
            match_config ? 'keywords' and
            match_config ? 'match_mode') or
        (match_type = 'regex' and
            match_config ? 'pattern') or
        (match_type = 'pattern' and
            match_config ? 'pattern_ids') or
        (match_type = 'default')
    )
);

-- Index for foreign key lookups (find rules by agent)
create index if not exists idx_routing_rules_agent
    on routing_rules(agent_name);

-- Table and column documentation
comment on table routing_rules is 'Rules for matching prompts to agents during routing';
comment on column routing_rules.id is 'Stable UUID identifier (rules may be renamed)';
comment on column routing_rules.name is 'Unique human-readable name (e.g., go-code-rule)';
comment on column routing_rules.priority is 'Evaluation priority (0-1000), higher values evaluated first';
comment on column routing_rules.agent_name is 'Target agent name (FK to agents table)';
comment on column routing_rules.match_type is 'Match algorithm: keyword, regex, pattern (semantic), or default';
comment on column routing_rules.match_config is 'Type-specific configuration as JSONB (structure varies by match_type)';
comment on column routing_rules.enabled is 'Whether this rule is active in routing evaluation';
comment on column routing_rules.created_at is 'Timestamp when the rule was created';
comment on column routing_rules.updated_at is 'Timestamp when the rule was last modified';
