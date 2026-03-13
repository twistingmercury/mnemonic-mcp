.PHONY: mnemonic help run stop

default: help

mnemonic: ## Run the build.sh script to build the mnemonic server
	LOCAL=1 ./src/mnemonic/build/build.sh

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nAvailable targets:\n"} /^[a-zA-Z0-9_-]+:.*##/ { printf "  %-12s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

run: ## Runs mnemonic using the latest build in the ghcr
	docker compose -f ./docker-compose-dev.yaml up -d

stop: ## Tears down the mnemonic infra that was started with 'make run' command
	docker compose -f ./docker-compose-dev.yaml down -v