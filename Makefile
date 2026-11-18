.DEFAULT_GOAL := help

.PHONY: help check

help: ## Show available commands
	@printf '%s\n' 'Available targets:'
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

check: ## Run repository-independent consistency checks
	@git diff --check
