SHELL := /bin/bash

DEMO_SCRIPTS := \
	scripts/demo-governance-loop.sh \
	scripts/demo-sprint2-cleanup.sh \
	scripts/demo-sprint3-mcp-policy.sh \
	scripts/demo-sprint4-credentials.sh \
	scripts/demo-sprint5-retry-config.sh \
	scripts/demo-sprint6-runtime-metrics.sh \
	scripts/demo-sprint7-credential-rotation.sh \
	scripts/demo-sprint8-management-audit.sh \
	scripts/demo-sprint9-route-policies.sh \
	scripts/demo-sprint10-route-policy-retry.sh \
	scripts/demo-sprint11-transactional-audit.sh

.PHONY: help check release-check test test-fresh vet build frontend-test frontend-build demo-scripts-lint test-postgres run demo-all

help:
	@printf 'AgentHarbor developer targets\n'
	@printf '\n'
	@printf '  make check              Run local backend, frontend, and demo-script checks\n'
	@printf '  make release-check      Run uncached release/merge readiness checks\n'
	@printf '  make test               Run Go tests\n'
	@printf '  make test-fresh         Run uncached Go tests\n'
	@printf '  make vet                Run go vet\n'
	@printf '  make build              Build Go packages\n'
	@printf '  make frontend-test      Run frontend unit tests\n'
	@printf '  make frontend-build     Build frontend assets\n'
	@printf '  make demo-scripts-lint  Syntax-check demo scripts\n'
	@printf '  make test-postgres      Run store tests using AGENT_HARBOR_TEST_DATABASE_URL\n'
	@printf '  make run                Start the local API server\n'
	@printf '  make demo-all           Run all demos against BASE_URL\n'

check: test vet build frontend-test frontend-build demo-scripts-lint

release-check: test-fresh vet build frontend-test frontend-build demo-scripts-lint

test:
	go test ./...

test-fresh:
	go test -count=1 ./...

vet:
	go vet ./...

build:
	go build ./...

frontend-test:
	pnpm --dir frontend test

frontend-build:
	pnpm --dir frontend build

demo-scripts-lint:
	bash -n $(DEMO_SCRIPTS) scripts/demo-all.sh

test-postgres:
	@if [[ -z "$$AGENT_HARBOR_TEST_DATABASE_URL" ]]; then \
		echo "AGENT_HARBOR_TEST_DATABASE_URL is required for make test-postgres" >&2; \
		exit 2; \
	fi
	go test ./internal/store -count=1

run:
	go run ./cmd/agent-harbor

demo-all:
	bash scripts/demo-all.sh
