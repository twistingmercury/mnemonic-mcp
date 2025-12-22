.DEFAULT_GOAL := help

.PHONY: help cognee-up cognee-down cognee-verify install-agents load-patterns validate-patterns

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make [target]\n\nTargets:\n"} /^[a-zA-Z_-]+:.*##/ {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

cognee-up: ## Start Cognee services and pull Ollama models
	@./scripts/setup-cognee.sh

cognee-down: ## Stop Cognee services and remove volumes
	cd memory-mcp && docker compose down -v --remove-orphans

cognee-verify: ## Verify Cognee infrastructure is ready
	@./scripts/verify-cognee.sh

install-agents: ## Install the agents in '~/.claude/agents/'
	@./scripts/install-agents.sh
	
update-global-conf: ## Create/append '~/.claude/CLAUDE.md' for agent delegation rules
	@ printf "NOT IMPLEMENTED\n"

load-patterns: ## Load patterns into Cognee
	@PATTERNS_DIR="$$(pwd)/claude-agents/patterns" ./scripts/load-patterns.sh

validate-patterns: ## Validate pattern metadata
	@PATTERNS_DIR="$$(pwd)/claude-agents/patterns" ./scripts/validate-metadata.sh

