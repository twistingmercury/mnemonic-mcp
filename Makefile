.PHONY: mnemonic ace help install-agents load-patterns enrich

default: help

install-agents: ## Installs the agent definitions locally to ~/.claude/agents for development purposes
	agents/workbench/scripts/01-install-agents.sh

enrich: ## Runs the cognify process for patterns that were loaded into cognee.
	agents/workbench/scripts/enrich-patterns.sh

mnemonic: ## Run the build.sh script to build the mnemonic server
	./src/mnemonic/build/build.sh
	docker system prune -f

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nAvailable targets:\n"} /^[a-zA-Z0-9_-]+:.*##/ { printf "  %-12s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)