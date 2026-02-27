.PHONY: mnemonic ace help install-agents load-patterns enrich

default: help

mnemonic: ## Run the build.sh script to build the mnemonic server
	./src/mnemonic/build/build.sh
	docker system prune -f

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nAvailable targets:\n"} /^[a-zA-Z0-9_-]+:.*##/ { printf "  %-12s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
