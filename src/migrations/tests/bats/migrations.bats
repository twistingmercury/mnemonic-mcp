#!/usr/bin/env bats
# BATS tests for PostgreSQL migration verification

# Connection details
export PGHOST=localhost
export PGPORT=5433
export PGDATABASE=mnemonic
export PGUSER=mnemonic
export PGPASSWORD=mnemonic_dev

# Helper: Run psql query and capture output
run_psql() {
    local query="$1"
    psql -t -A -c "$query" 2>&1
}

# Helper: Count rows from psql query result
count_result() {
    local query="$1"
    local result
    result=$(run_psql "$query")
    printf '%s' "$result" | wc -l | tr -d ' '
}

@test "extension uuid-ossp exists" {
    local query="SELECT 1 FROM pg_extension WHERE extname = 'uuid-ossp';"
    local result
    result=$(run_psql "$query")

    [ -n "$result" ]
}

@test "extension vector exists" {
    local query="SELECT 1 FROM pg_extension WHERE extname = 'vector';"
    local result
    result=$(run_psql "$query")

    [ -n "$result" ]
}

@test "table agents exists" {
    local query="SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'agents';"
    local result
    result=$(run_psql "$query")

    [ -n "$result" ]
}

@test "table patterns exists" {
    local query="SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'patterns';"
    local result
    result=$(run_psql "$query")

    [ -n "$result" ]
}

@test "table pattern_agent_associations exists" {
    local query="SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'pattern_agent_associations';"
    local result
    result=$(run_psql "$query")

    [ -n "$result" ]
}

@test "table routing_rules exists" {
    local query="SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'routing_rules';"
    local result
    result=$(run_psql "$query")

    [ -n "$result" ]
}

@test "table enrichment_jobs exists" {
    local query="SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'enrichment_jobs';"
    local result
    result=$(run_psql "$query")

    [ -n "$result" ]
}

@test "expected number of indexes exist" {
    # Count all indexes on our tables (excluding primary key indexes which are constraints)
    # Expected indexes from migrations:
    #   patterns:
    #     - idx_patterns_embedding_cosine (from 003)
    #     - idx_patterns_enrichment_status (from 003)
    #     - idx_patterns_search (from 003)
    #     - idx_patterns_enriched (from 007)
    #     - idx_patterns_tags (from 007)
    #   routing_rules:
    #     - idx_routing_rules_enabled_priority (from 007)
    #   enrichment_jobs:
    #     - idx_enrichment_jobs_pending (from 007)
    #     - idx_enrichment_jobs_processing (from 007)
    #   pattern_agent_associations:
    #     - idx_pattern_agent_associations_agent_id (assumed from schema)
    #     - idx_pattern_agent_associations_pattern_id (assumed from schema)
    #
    # Total expected: 10 named indexes
    # Note: This excludes automatic primary key and unique constraint indexes

    local query="SELECT COUNT(*) FROM pg_indexes WHERE schemaname = 'public' AND indexname LIKE 'idx_%';"
    local count
    count=$(run_psql "$query")

    # We expect at least 8 explicitly named indexes from the migrations we read
    # (patterns: 5, routing_rules: 1, enrichment_jobs: 2)
    # Pattern associations may have additional indexes
    [ "$count" -ge 8 ]
}

# =============================================================================
# Down Migration Tests
# These tests run down migrations in reverse order (007 → 001) to verify
# proper rollback functionality. Each test runs the migration and verifies
# the expected objects were removed.
# =============================================================================

@test "down migration 007 - drops performance indexes" {
    # Find the migrations directory relative to this test file
    # BATS_TEST_FILENAME is bats/migrations.bats, we need to go up two levels to migrations/
    local test_dir
    test_dir="$(cd "$(dirname "${BATS_TEST_FILENAME}")/../.." && pwd)"
    local mig_dir="${test_dir}/postgres/down"
    local migration_file="${mig_dir}/007_create_performance_indexes.sql"

    # Verify migration file exists
    [ -f "$migration_file" ]

    # Run down migration
    psql -f "$migration_file" >/dev/null 2>&1

    # Verify idx_enrichment_jobs_processing is dropped
    local query="SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_enrichment_jobs_processing';"
    local result
    result=$(run_psql "$query")
    [ -z "$result" ]

    # Verify idx_enrichment_jobs_pending is dropped
    query="SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_enrichment_jobs_pending';"
    result=$(run_psql "$query")
    [ -z "$result" ]

    # Verify idx_patterns_tags is dropped
    query="SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_patterns_tags';"
    result=$(run_psql "$query")
    [ -z "$result" ]

    # Verify idx_patterns_enriched is dropped
    query="SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_patterns_enriched';"
    result=$(run_psql "$query")
    [ -z "$result" ]

    # Verify idx_routing_rules_enabled_priority is dropped
    query="SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_routing_rules_enabled_priority';"
    result=$(run_psql "$query")
    [ -z "$result" ]
}

@test "down migration 006 - drops enrichment_jobs table" {
    local test_dir
    test_dir="$(cd "$(dirname "${BATS_TEST_FILENAME}")/../.." && pwd)"
    local mig_dir="${test_dir}/postgres/down"
    local migration_file="${mig_dir}/006_create_enrichment_jobs.sql"

    # Verify migration file exists
    [ -f "$migration_file" ]

    # Run down migration
    psql -f "$migration_file" >/dev/null 2>&1

    # Verify enrichment_jobs table is dropped
    local query="SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'enrichment_jobs';"
    local result
    result=$(run_psql "$query")
    [ -z "$result" ]

    # Verify idx_enrichment_jobs_pattern is dropped (if it wasn't already)
    query="SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_enrichment_jobs_pattern';"
    result=$(run_psql "$query")
    [ -z "$result" ]
}

@test "down migration 005 - drops routing_rules table" {
    local test_dir
    test_dir="$(cd "$(dirname "${BATS_TEST_FILENAME}")/../.." && pwd)"
    local mig_dir="${test_dir}/postgres/down"
    local migration_file="${mig_dir}/005_create_routing_rules.sql"

    # Verify migration file exists
    [ -f "$migration_file" ]

    # Run down migration
    psql -f "$migration_file" >/dev/null 2>&1

    # Verify routing_rules table is dropped
    local query="SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'routing_rules';"
    local result
    result=$(run_psql "$query")
    [ -z "$result" ]

    # Verify idx_routing_rules_agent is dropped (if it wasn't already)
    query="SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_routing_rules_agent';"
    result=$(run_psql "$query")
    [ -z "$result" ]
}

@test "down migration 004 - drops pattern_agent_associations table" {
    local test_dir
    test_dir="$(cd "$(dirname "${BATS_TEST_FILENAME}")/../.." && pwd)"
    local mig_dir="${test_dir}/postgres/down"
    local migration_file="${mig_dir}/004_create_pattern_agent_associations.sql"

    # Verify migration file exists
    [ -f "$migration_file" ]

    # Run down migration
    psql -f "$migration_file" >/dev/null 2>&1

    # Verify pattern_agent_associations table is dropped
    local query="SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'pattern_agent_associations';"
    local result
    result=$(run_psql "$query")
    [ -z "$result" ]

    # Verify idx_pattern_agent_assoc_agent is dropped
    query="SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_pattern_agent_assoc_agent';"
    result=$(run_psql "$query")
    [ -z "$result" ]

    # Verify idx_pattern_agent_assoc_pattern is dropped
    query="SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_pattern_agent_assoc_pattern';"
    result=$(run_psql "$query")
    [ -z "$result" ]
}

@test "down migration 003 - drops patterns table" {
    local test_dir
    test_dir="$(cd "$(dirname "${BATS_TEST_FILENAME}")/../.." && pwd)"
    local mig_dir="${test_dir}/postgres/down"
    local migration_file="${mig_dir}/003_create_patterns.sql"

    # Verify migration file exists
    [ -f "$migration_file" ]

    # Run down migration
    psql -f "$migration_file" >/dev/null 2>&1

    # Verify patterns table is dropped
    local query="SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'patterns';"
    local result
    result=$(run_psql "$query")
    [ -z "$result" ]
}

@test "down migration 002 - drops agents table" {
    local test_dir
    test_dir="$(cd "$(dirname "${BATS_TEST_FILENAME}")/../.." && pwd)"
    local mig_dir="${test_dir}/postgres/down"
    local migration_file="${mig_dir}/002_create_agents.sql"

    # Verify migration file exists
    [ -f "$migration_file" ]

    # Run down migration
    psql -f "$migration_file" >/dev/null 2>&1

    # Verify agents table is dropped
    local query="SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'agents';"
    local result
    result=$(run_psql "$query")
    [ -z "$result" ]
}

@test "down migration 001 - extensions remain (not dropped)" {
    local test_dir
    test_dir="$(cd "$(dirname "${BATS_TEST_FILENAME}")/../.." && pwd)"
    local mig_dir="${test_dir}/postgres/down"
    local migration_file="${mig_dir}/001_extensions_and_functions.sql"

    # Verify migration file exists
    [ -f "$migration_file" ]

    # Run down migration (this is a no-op per the migration comments)
    psql -f "$migration_file" >/dev/null 2>&1

    # Verify uuid-ossp extension still exists (should not be dropped)
    local query="SELECT 1 FROM pg_extension WHERE extname = 'uuid-ossp';"
    local result
    result=$(run_psql "$query")
    [ -n "$result" ]

    # Verify vector extension still exists (should not be dropped)
    query="SELECT 1 FROM pg_extension WHERE extname = 'vector';"
    result=$(run_psql "$query")
    [ -n "$result" ]
}
