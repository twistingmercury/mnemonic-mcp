-- src/migrations/postgres/up/002_create_agents.sql
-- Creates the agents table for storing agent definitions.
-- Part of Mnemonic MVP
--
-- Dependencies: 001_extensions_and_functions (for extensions)
--
-- Agents are the core execution targets for routing decisions.
-- The name field serves as a natural primary key and must follow
-- Kubernetes-style naming conventions (lowercase with hyphens).
--
-- Note: updated_at is managed by the application layer (Go repository)
-- rather than database triggers for better control and testability.

create table if not exists agents (
    -- Primary key: lowercase-with-hyphens format, URL-safe
    -- Examples: go-software-agent, data-architect, api-gateway
    name varchar(64) primary key,

    -- Human-readable description of the agent's purpose
    description varchar(500) not null,

    -- System prompt content (up to 50KB)
    -- This is the full system prompt provided to the LLM
    system_prompt text not null,

    -- Model preference: sonnet, opus, haiku, or inherit from caller
    model varchar(20) not null default 'inherit',

    -- Allowed MCP tools (JSON array of tool names)
    -- Example: ["read_file", "write_file", "execute_command"]
    allowed_tools jsonb not null default '[]'::jsonb,

    -- Keywords for fast routing (denormalized from routing_rules)
    -- Example: ["go", "golang", "backend"]
    routing_keywords jsonb not null default '[]'::jsonb,

    -- Audit timestamps
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),

    -- Constraints

    -- Name must be lowercase letters, numbers, and hyphens only
    -- Must start with a letter (not a number or hyphen)
    constraint agents_name_format
        check (name ~ '^[a-z][a-z0-9-]*$'),

    -- Model must be one of the valid Claude model tiers or inherit
    constraint agents_model_valid
        check (model in ('sonnet', 'opus', 'haiku', 'inherit')),

    -- System prompt has a maximum length of 50KB (51200 bytes)
    constraint agents_system_prompt_length
        check (length(system_prompt) <= 51200),

    -- allowed_tools must be a JSON array
    constraint agents_allowed_tools_array
        check (jsonb_typeof(allowed_tools) = 'array'),

    -- routing_keywords must be a JSON array
    constraint agents_routing_keywords_array
        check (jsonb_typeof(routing_keywords) = 'array')
);

-- Table and column documentation
comment on table agents is 'Agent definitions for the routing system';
comment on column agents.name is 'Unique identifier, lowercase-with-hyphens format (e.g., go-software-agent)';
comment on column agents.description is 'Human-readable description of the agent purpose';
comment on column agents.system_prompt is 'Full system prompt provided to the LLM (up to 50KB)';
comment on column agents.model is 'Claude model preference: sonnet, opus, haiku, or inherit from caller';
comment on column agents.allowed_tools is 'JSON array of MCP tool names this agent can use';
comment on column agents.routing_keywords is 'Denormalized keywords for fast routing lookups';
comment on column agents.created_at is 'Timestamp when the agent was created';
comment on column agents.updated_at is 'Timestamp when the agent was last modified';
