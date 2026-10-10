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
# The API diff gate installs apidiff at this golang.org/x/exp version into a
# temporary directory; it is never added to go.mod
# (docs/decisions/0014-PLAN-api-policy-gates.md Step 4).
APIDIFF_VERSION ?= v0.0.0-20261007180756-3d68b386da03
# The scripts under scripts/ are Python, the standard library only
# (docs/decisions/0017-MADR-python-repository-scripts.md).
PYTHON ?= python3

.PHONY: all help test test-sum fmt vet lint modernize tidy tidy-check vuln fuzz apicheck examples pre-add-check release-check

all: help

# Every gate runs once per module, in the module's directory, with
# GOWORK=off, so that a module builds from its own go.mod and published
# versions only (docs/decisions/0010-MADR-nested-adapter-modules.md §4).
# scripts/go-modules.py lists the modules go.work names; a failure to list
# them fails the target, rather than looping over nothing.
MODULES_CMD = $(PYTHON) ./scripts/go-modules.py

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

# Fuzz each fuzz target for FUZZTIME; go test -fuzz takes one target per run
# (docs/decisions/0002-PLAN-multi-pane-workspace-layouts.md Step 5,
# docs/decisions/0006-PLAN-command-registry.md Step 2).
FUZZTIME ?= 20s
# The packages are found, not listed: every package whose tests declare a
# fuzz target, per module (docs/decisions/0014-PLAN-hardening.md Step 9).
fuzz: ## Fuzzes every fuzz target, found in every package of every module, for FUZZTIME each (default 20s)
	$(call each_module,$(PYTHON) $(CURDIR)/scripts/go-fuzz.py -a -t $(FUZZTIME) -m 1 ./...)

# The API diff gate: an incompatible change since the previous tag fails unless
# scripts/apicheck.allow lists it (docs/decisions/0014-PLAN-api-policy-gates.md Step 4).
apicheck: ## Fails on an incompatible API change since the previous tag, per module
	@APIDIFF_VERSION=$(APIDIFF_VERSION) $(PYTHON) ./scripts/go-apicheck.py

# The framework examples under testdata/frameworks, built against the tree in
# a temporary module of their own, run, and compared with the guides'
# excerpts (docs/decisions/0014-PLAN-api-policy-gates.md Step 5).
examples: ## Builds and runs the framework examples in a temporary module
	@$(PYTHON) ./scripts/go-examples.py

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
