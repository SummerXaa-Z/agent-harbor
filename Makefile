SHELL := /bin/bash

PNPM ?= ./scripts/pnpm.sh

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
	scripts/scenario-tenant-permission-center.sh \
	scripts/demo.sh \
	scripts/scenario-tenant-access-profile.sh

SCENARIO_SCRIPT_LIBS := \
	scripts/lib/ports.sh \
	scripts/pnpm.sh

.PHONY: help check release-check fmt gofmt-check test test-fresh vet build frontend-deps frontend-test frontend-build real-mcp-deps makefile-targets-test scenario-scripts-lint github-config-lint test-postgres run mock-mcp real-mcp demo core-journey scenario-permission-package-approval ai-admin-browser-journey web-console-production-journey production-hardening scenario-admin-tenant-boundary scenario-admin-access-management scenario-tenant-permission-center scenario-all

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
	@printf '  make scenario-tenant-permission-center Run tenant permission center projection gate\n'
	@printf '  make scenario-all          Run all scenarios against BASE_URL\n'

check: gofmt-check test vet build makefile-targets-test frontend-test frontend-build scenario-scripts-lint github-config-lint

release-check: gofmt-check test-fresh vet build production-hardening scenario-permission-package-approval ai-admin-browser-journey web-console-production-journey scenario-admin-tenant-boundary scenario-admin-access-management scenario-tenant-permission-center makefile-targets-test frontend-test frontend-build scenario-scripts-lint github-config-lint

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
	$(PNPM) --dir frontend install --frozen-lockfile

frontend-test: frontend-deps
	$(PNPM) --dir frontend test

frontend-build: frontend-deps
	$(PNPM) --dir frontend build

real-mcp-deps:
	$(PNPM) --dir scripts/real-mcp install --frozen-lockfile

makefile-targets-test:
	bash tests/makefile_targets_test.sh

scenario-scripts-lint:
	bash -n $(SCENARIO_SCRIPTS) $(SCENARIO_SCRIPT_LIBS) scripts/scenario-all.sh

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
	$(PNPM) --dir scripts/real-mcp start

demo: scripts/demo.sh
	scripts/demo.sh

core-journey:
	bash scripts/scenario-core-journey.sh

scenario-permission-package-approval:
	@if [[ -n "$${BASE_URL:-}" ]]; then \
		bash scripts/scenario-permission-package-approval.sh; \
	else \
		source scripts/lib/ports.sh; \
		run_id="permission-package-approval-$$(date +%Y%m%d%H%M%S)"; \
		api_host="$${AGENT_HARBOR_PERMISSION_PACKAGE_API_HOST:-127.0.0.1}"; \
		api_port="$${AGENT_HARBOR_PERMISSION_PACKAGE_API_PORT:-9197}"; \
		api_addr="$${AGENT_HARBOR_PERMISSION_PACKAGE_API_ADDR:-$${api_host}:$${api_port}}"; \
		base_url="http://$${api_host}:$${api_port}"; \
		mcp_port="$${MOCK_MCP_PORT:-8797}"; \
		log_dir="$${TMPDIR:-/tmp}/agent-harbor-permission-package-approval-$${run_id}"; \
		requester_key="permission-package-requester-key-$${run_id}"; \
		reviewer_key="permission-package-reviewer-key-$${run_id}"; \
		assert_port_free "API" "$$api_port"; \
		assert_port_free "MCP" "$$mcp_port"; \
		mkdir -p "$$log_dir"; \
		cleanup() { \
			if [[ -n "$${api_pid:-}" ]]; then \
				kill "$$api_pid" >/dev/null 2>&1 || true; \
				wait "$$api_pid" >/dev/null 2>&1 || true; \
			fi; \
		}; \
		trap cleanup EXIT; \
		AGENT_HARBOR_ADDR="$$api_addr" \
		AGENT_HARBOR_ADMIN_IDENTITIES="requester=$$requester_key;security-reviewer=$$reviewer_key" \
		AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true \
			go run ./cmd/agent-harbor >"$$log_dir/api.log" 2>&1 & \
		api_pid="$$!"; \
		for _ in $$(seq 1 80); do \
			if curl -fsS "$$base_url/healthz" >/dev/null 2>&1; then \
				break; \
			fi; \
			sleep 0.25; \
		done; \
		if ! curl -fsS "$$base_url/healthz" >/dev/null 2>&1; then \
			echo "AgentHarbor API did not become ready for scenario-permission-package-approval" >&2; \
			tail -80 "$$log_dir/api.log" >&2 || true; \
			exit 1; \
		fi; \
		BASE_URL="$$base_url" \
		REQUESTER_ADMIN_KEY="$$requester_key" \
		REVIEWER_ADMIN_KEY="$$reviewer_key" \
		ADMIN_KEY="$$requester_key" \
		RUN_ID="$$run_id" \
		MOCK_MCP_PORT="$$mcp_port" \
			bash scripts/scenario-permission-package-approval.sh; \
	fi

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

.PHONY: scenario-tenant-permission-center
scenario-tenant-permission-center: build
	bash scripts/scenario-tenant-permission-center.sh

scenario-all:
	bash scripts/scenario-all.sh
