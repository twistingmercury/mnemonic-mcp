.DEFAULT_GOAL := help

.PHONY: help cognee-up cognee-down cognee-verify install-agents load-patterns validate-patterns

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make [target]\n\nTargets:\n"} /^[a-zA-Z_-]+:.*##/ {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)


