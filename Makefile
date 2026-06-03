SHELL := /bin/bash

GO_FILES := $(shell git ls-files '*.go')

SCENARIO_SCRIPTS := \
	scripts/scenario-core-journey.sh \
	scripts/scenario-governance-loop.sh \
	scripts/scenario-registry-cleanup.sh \
	scripts/scenario-mcp-policy.sh \
	scripts/scenario-credential-redaction.sh \
	scripts/scenario-retry-config.sh \
	scripts/scenario-runtime-metrics.sh \
	scripts/scenario-credential-rotation.sh \
	scripts/scenario-management-audit.sh \
	scripts/scenario-route-policies.sh \
	scripts/scenario-route-policy-retry.sh \
	scripts/scenario-transactional-audit.sh \
	scripts/scenario-mcp-capability-governance.sh \
	scripts/scenario-data-permission-enforcement.sh \
	scripts/scenario-tenant-hierarchy.sh \
	scripts/demo.sh \
	scripts/scenario-tenant-access-profile.sh

.PHONY: help check release-check fmt gofmt-check test test-fresh vet build frontend-deps frontend-test frontend-build makefile-targets-test scenario-scripts-lint github-config-lint test-postgres run mock-mcp demo core-journey scenario-all

help:
	@printf 'AgentHarbor developer targets\n'
	@printf '\n'
	@printf '  make check                 Run local backend, frontend, and scenario-script checks\n'
	@printf '  make release-check         Run uncached release/merge readiness checks\n'
	@printf '  make fmt                   Format Go files with gofmt\n'
	@printf '  make gofmt-check           Verify Go files are gofmt-formatted\n'
	@printf '  make test                  Run Go tests\n'
	@printf '  make test-fresh            Run uncached Go tests\n'
	@printf '  make vet                   Run go vet\n'
	@printf '  make build                 Build Go packages\n'
	@printf '  make frontend-deps         Install pinned frontend dependencies\n'
	@printf '  make frontend-test         Run frontend unit tests\n'
	@printf '  make frontend-build        Build frontend assets\n'
	@printf '  make makefile-targets-test Verify Makefile release-gate dependencies\n'
	@printf '  make scenario-scripts-lint Syntax-check scenario scripts\n'
	@printf '  make github-config-lint    Parse-check GitHub YAML configuration\n'
	@printf '  make test-postgres         Run store tests using AGENT_HARBOR_TEST_DATABASE_URL\n'
	@printf '  make run                   Start the local API server\n'
	@printf '  make mock-mcp              Start the local mock MCP server for console demos\n'
	@printf '  make demo                  Start API, mock MCP, and web console for first-run evaluation\n'
	@printf '  make core-journey          Run the 10-minute local core journey scenario\n'
	@printf '  make scenario-all          Run all scenarios against BASE_URL\n'

check: gofmt-check test vet build makefile-targets-test frontend-test frontend-build scenario-scripts-lint github-config-lint

release-check: gofmt-check test-fresh vet build makefile-targets-test frontend-test frontend-build scenario-scripts-lint github-config-lint

fmt:
	gofmt -w $(GO_FILES)

gofmt-check:
	@files="$$(gofmt -l $(GO_FILES))"; \
	if [[ -n "$$files" ]]; then \
		echo "Go files need gofmt:"; \
		echo "$$files"; \
		exit 1; \
	fi

test:
	go test ./...

test-fresh:
	go test -count=1 ./...

vet:
	go vet ./...

build:
	go build ./...

frontend-deps:
	pnpm --dir frontend install --frozen-lockfile

frontend-test: frontend-deps
	pnpm --dir frontend test

frontend-build: frontend-deps
	pnpm --dir frontend build

makefile-targets-test:
	bash tests/makefile_targets_test.sh

scenario-scripts-lint:
	bash -n $(SCENARIO_SCRIPTS) scripts/scenario-all.sh

github-config-lint:
	ruby -e 'require "yaml"; Dir[".github/**/*.yml"].sort.each { |file| YAML.load_file(file); puts "#{file} ok" }'

test-postgres:
	@if [[ -z "$$AGENT_HARBOR_TEST_DATABASE_URL" ]]; then \
		echo "AGENT_HARBOR_TEST_DATABASE_URL is required for make test-postgres" >&2; \
		exit 2; \
	fi
	go test ./internal/store -count=1

run:
	go run ./cmd/agent-harbor

mock-mcp:
	scripts/mock-mcp-server.py --host "$${MOCK_MCP_HOST:-127.0.0.1}" --port "$${MOCK_MCP_PORT:-8787}"

demo: frontend-deps scripts/demo.sh
	scripts/demo.sh

core-journey:
	bash scripts/scenario-core-journey.sh

scenario-all:
	bash scripts/scenario-all.sh
