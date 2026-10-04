# go-tui-lib — shared Go library of terminal-UI packages on Charm v2
# (no binary packaging targets).
#
# Prefer the user's Go toolchain install (go install ...), then PATH.
# go env prints Windows paths with backslashes, which a recipe's shell
# drops; forward slashes work on every host
# (docs/decisions/0010-PLAN-nested-adapter-modules.md deviation D1).
GOPATH_BIN    := $(subst \,/,$(shell go env GOPATH))/bin
GOBIN         := $(subst \,/,$(shell go env GOBIN))
GOLANGCI_LINT ?= $(GOPATH_BIN)/golangci-lint
GOVULNCHECK   ?= $(or $(wildcard $(GOBIN)/govulncheck),$(GOPATH_BIN)/govulncheck,$(shell command -v govulncheck 2>/dev/null))
GOTESTSUM     ?= $(or $(wildcard $(GOBIN)/gotestsum),$(GOPATH_BIN)/gotestsum,$(shell command -v gotestsum 2>/dev/null))
FLEET_LINT_CFG := .golangci.yml

.PHONY: all help test test-sum fmt vet lint modernize tidy tidy-check vuln fuzz pre-add-check release-check

all: help

# Every gate runs once per module, in the module's directory, with
# GOWORK=off, so that a module builds from its own go.mod and published
# versions only (docs/decisions/0010-MADR-nested-adapter-modules.md §4).
# scripts/go-modules.sh lists the modules go.work names; a failure to list
# them fails the target, rather than looping over nothing.
MODULES_CMD := ./scripts/go-modules.sh

# each_module runs $(1) in every module's directory with GOWORK=off.
define each_module
	@mods="$$($(MODULES_CMD))" || exit 1; status=0; \
	for m in $$mods; do \
		echo "== module $$m"; \
		(cd "$$m" && GOWORK=off $(1)) || status=1; \
	done; exit $$status
endef

test: ## Runs all tests, per module
	$(call each_module,go test ./...)

test-sum: ## Runs tests via gotestsum (opt-in; requires gotestsum on PATH/GOBIN)
	@if [ -z "$(GOTESTSUM)" ] || [ ! -x "$(GOTESTSUM)" ]; then \
		echo "gotestsum not found. Install: go install gotest.tools/gotestsum@latest"; \
		exit 1; \
	fi
	$(GOTESTSUM) --format testname -- ./...

fmt: ## Formats all Go source files
	go fmt ./...

vet: ## Runs go vet, per module
	$(call each_module,go vet ./...)

# Lint every target the code builds for, with cgo off so a cross-target run
# never needs a C toolchain for that target
# (go-selfupdate-lib docs/decisions/0002-MADR-rehome-selfupdate-from-mcplib.md §5).
LINT_GOOS := linux darwin windows

lint: modernize ## Runs make modernize, then golangci-lint with fleet config for linux, darwin and windows, per module
	@if [ ! -x "$(GOLANGCI_LINT)" ]; then \
		echo "golangci-lint not found at $(GOLANGCI_LINT)"; \
		echo "Install: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest"; \
		exit 1; \
	fi
	@mods="$$($(MODULES_CMD))" || exit 1; status=0; root="$$(git rev-parse --show-toplevel)"; \
	for m in $$mods; do for os in $(LINT_GOOS); do \
		echo "golangci-lint (module $$m, GOOS=$$os)"; \
		(cd "$$m" && CGO_ENABLED=0 GOOS=$$os GOWORK=off $(GOLANGCI_LINT) run -c "$$root/$(FLEET_LINT_CFG)" ./...) || { echo "golangci-lint failed for module $$m, GOOS=$$os"; status=1; }; \
	done; done; exit $$status

# go fix -diff prints what go fix would change and exits 1 when that is
# anything (docs/decisions/0002-PLAN-harden-workspace-v0-1-1.md Step 7).
modernize: ## Fails on any go fix suggestion, for linux, darwin and windows, per module
	@mods="$$($(MODULES_CMD))" || exit 1; status=0; \
	for m in $$mods; do for os in $(LINT_GOOS); do \
		echo "go fix -diff (module $$m, GOOS=$$os)"; \
		(cd "$$m" && CGO_ENABLED=0 GOOS=$$os GOWORK=off go fix -diff ./...) || { echo "go fix has suggestions for module $$m, GOOS=$$os"; status=1; }; \
	done; done; exit $$status

tidy: ## Runs go mod tidy
	go mod tidy

tidy-check: ## Fails when go mod tidy would change a module's go.mod or go.sum, per module
	$(call each_module,go mod tidy -diff)

vuln: ## Runs govulncheck (opt-in; requires govulncheck on PATH/GOBIN)
	@if [ -z "$(GOVULNCHECK)" ] || [ ! -x "$(GOVULNCHECK)" ]; then \
		echo "govulncheck not found. Install: go install golang.org/x/vuln/cmd/govulncheck@v1.8.0"; \
		exit 1; \
	fi
	$(call each_module,$(GOVULNCHECK) ./...)

# Fuzz each layout fuzz target for FUZZTIME; go test -fuzz takes one target
# per run (docs/decisions/0002-PLAN-multi-pane-workspace-layouts.md Step 5).
FUZZTIME ?= 20s
fuzz: ## Fuzzes every layout fuzz target for FUZZTIME each (default 20s)
	@./scripts/go-fuzz.sh -t $(FUZZTIME) -m 1 ./layout

# The pre-add rule (AGENTS.md). scripts/go-precheck.sh is the one
# implementation; the agent gate at `git commit` runs the same file.
# Pass FILES=... to check specific files instead of every tracked Go file.
FILES ?=
pre-add-check: ## Runs the pre-add checks (gofmt, golangci-lint, vet, test, tidy, govulncheck), per module
	@./scripts/go-precheck.sh $(FILES)

# Before a tag: the pre-add checks over every module, with no file list.
release-check: ## Runs the pre-add checks over every module and file
	@./scripts/go-precheck.sh

help: ## Displays this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'
