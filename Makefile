SHELL := /bin/bash

GO_FILES := $(shell git ls-files '*.go')

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
	scripts/demo-sprint11-transactional-audit.sh \
	scripts/demo-sprint12-mcp-capability-governance.sh \
	scripts/demo-sprint13-data-permission-enforcement.sh \
	scripts/demo-sprint14-tenant-hierarchy.sh

.PHONY: help check release-check fmt gofmt-check test test-fresh vet build frontend-test frontend-build demo-scripts-lint github-config-lint test-postgres run demo-all

help:
	@printf 'AgentHarbor developer targets\n'
	@printf '\n'
	@printf '  make check              Run local backend, frontend, and demo-script checks\n'
	@printf '  make release-check      Run uncached release/merge readiness checks\n'
	@printf '  make fmt                Format Go files with gofmt\n'
	@printf '  make gofmt-check        Verify Go files are gofmt-formatted\n'
	@printf '  make test               Run Go tests\n'
	@printf '  make test-fresh         Run uncached Go tests\n'
	@printf '  make vet                Run go vet\n'
	@printf '  make build              Build Go packages\n'
	@printf '  make frontend-test      Run frontend unit tests\n'
	@printf '  make frontend-build     Build frontend assets\n'
	@printf '  make demo-scripts-lint  Syntax-check demo scripts\n'
	@printf '  make github-config-lint Parse-check GitHub YAML configuration\n'
	@printf '  make test-postgres      Run store tests using AGENT_HARBOR_TEST_DATABASE_URL\n'
	@printf '  make run                Start the local API server\n'
	@printf '  make demo-all           Run all demos against BASE_URL\n'

check: gofmt-check test vet build frontend-test frontend-build demo-scripts-lint github-config-lint

release-check: gofmt-check test-fresh vet build frontend-test frontend-build demo-scripts-lint github-config-lint

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

frontend-test:
	pnpm --dir frontend test

frontend-build:
	pnpm --dir frontend build

demo-scripts-lint:
	bash -n $(DEMO_SCRIPTS) scripts/demo-all.sh

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

demo-all:
	bash scripts/demo-all.sh
