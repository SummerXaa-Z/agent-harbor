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
	scripts/scenario-permission-package-approval.sh \
	scripts/scenario-ai-admin-browser-journey.sh \
	scripts/scenario-web-console-production-journey.sh \
	scripts/scenario-production-hardening.sh \
	scripts/scenario-admin-tenant-boundary.sh \
	scripts/scenario-admin-access-management.sh \
	scripts/demo.sh \
	scripts/scenario-tenant-access-profile.sh

.PHONY: help check release-check fmt gofmt-check test test-fresh vet build frontend-deps frontend-test frontend-build real-mcp-deps makefile-targets-test scenario-scripts-lint github-config-lint test-postgres run mock-mcp real-mcp demo core-journey scenario-permission-package-approval ai-admin-browser-journey web-console-production-journey production-hardening scenario-admin-tenant-boundary scenario-admin-access-management scenario-all

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
	@printf '  make mock-mcp              Start the dependency-free mock MCP server for low-level tests\n'
	@printf '  make real-mcp              Start the local official SDK MCP demo server\n'
	@printf '  make demo                  Start API, real MCP, and web console for first-run evaluation\n'
	@printf '  make core-journey          Run the 10-minute local core journey scenario\n'
	@printf '  make scenario-permission-package-approval Run the local approval-required permission package scenario\n'
	@printf '  make ai-admin-browser-journey Run the browser-facing AI Admin approval journey release gate\n'
	@printf '  make web-console-production-journey Run the web console production journey smoke gate\n'
	@printf '  make production-hardening  Run the production safety baseline gate\n'
	@printf '  make scenario-admin-tenant-boundary Run scoped admin tenant/workspace boundary gate\n'
	@printf '  make scenario-admin-access-management Run managed administrator lifecycle gate\n'
	@printf '  make scenario-all          Run all scenarios against BASE_URL\n'

check: gofmt-check test vet build makefile-targets-test frontend-test frontend-build scenario-scripts-lint github-config-lint

release-check: gofmt-check test-fresh vet build production-hardening web-console-production-journey scenario-admin-tenant-boundary scenario-admin-access-management makefile-targets-test frontend-test frontend-build scenario-scripts-lint github-config-lint

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

real-mcp-deps:
	pnpm --dir scripts/real-mcp install --frozen-lockfile

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

real-mcp: real-mcp-deps
	pnpm --dir scripts/real-mcp start

demo: frontend-deps real-mcp-deps scripts/demo.sh
	scripts/demo.sh

core-journey:
	bash scripts/scenario-core-journey.sh

scenario-permission-package-approval:
	bash scripts/scenario-permission-package-approval.sh

ai-admin-browser-journey: frontend-deps
	bash scripts/scenario-ai-admin-browser-journey.sh

web-console-production-journey: frontend-deps real-mcp-deps
	bash scripts/scenario-web-console-production-journey.sh

production-hardening:
	bash scripts/scenario-production-hardening.sh

scenario-admin-tenant-boundary:
	bash scripts/scenario-admin-tenant-boundary.sh

scenario-admin-access-management:
	bash scripts/scenario-admin-access-management.sh

scenario-all:
	bash scripts/scenario-all.sh
