-- src/mnemonic/migrations/postgres/down/005_create_routing_rules.sql
-- Reverses: Creates the routing_rules table for prompt-to-agent matching.
--
-- This migration drops the routing_rules table and its indexes.
-- No other tables depend on routing_rules, so it can be safely dropped.

drop index if exists idx_routing_rules_agent;
drop table if exists routing_rules;
