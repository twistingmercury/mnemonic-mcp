.PHONY: mnemonic ace help install-agents load-patterns enrich

default: help

install-agents: ## Intalls the agent definitions locally to ~/.claude/agents for development purposes
	scripts/install-agents.sh

load-patterns: ## loads the agent patterns into Cognee for local development purposes
	scripts/load-patterns.sh

enrich: ## Runs the cognify process for patterns that were loaded into cognee.
	scripts/cognify-patterns.sh

mnemonic: ## Run the build.sh script to build the mnemonic server
	./src/mnemonic/build/build.sh
	docker system prune -f

ace: ## Run the build.sh script to build the ACE cli tool
	@echo "make command ace not implemented"

agent-rules: ## Run the install-global-agent-rules.sh to help guide Claude on how to use the agents
	scripts/install-global-agent-rules.sh


help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nAvailable targets:\n"} /^[a-zA-Z0-9_-]+:.*##/ { printf "  %-12s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)