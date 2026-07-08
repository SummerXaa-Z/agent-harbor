#!/usr/bin/env bash
set -euo pipefail

assert_target_depends_on() {
  local target="$1"
  local dependency="$2"
  if ! awk -v target="$target" -v dependency="$dependency" '
    $0 ~ "^[[:space:]]*" target ":" {
      split($0, parts, ":")
      split(parts[2], deps, /[[:space:]]+/)
      for (i in deps) {
        if (deps[i] == dependency) {
          found = 1
        }
      }
    }
    END { exit found ? 0 : 1 }
  ' Makefile; then
    echo "expected target ${target} to depend on ${dependency}" >&2
    exit 1
  fi
}

assert_target_exists() {
  local target="$1"
  if ! grep -Eq "^[[:space:]]*${target}:" Makefile; then
    echo "expected target ${target} to exist" >&2
    exit 1
  fi
}

assert_file_contains() {
  local file="$1"
  local needle="$2"
  if ! grep -Fq "$needle" "$file"; then
    echo "expected ${file} to contain ${needle}" >&2
    exit 1
  fi
}

assert_target_depends_on "frontend-test" "frontend-deps"
assert_target_depends_on "frontend-build" "frontend-deps"
assert_target_depends_on "ai-admin-browser-journey" "frontend-deps"
assert_target_depends_on "release-check" "production-hardening"
assert_target_depends_on "release-check" "ai-admin-browser-journey"
assert_target_depends_on "web-console-production-journey" "frontend-deps"
assert_target_depends_on "web-console-production-journey" "real-mcp-deps"
assert_target_depends_on "release-check" "web-console-production-journey"
assert_target_exists "scenario-permission-package-approval"
assert_target_depends_on "release-check" "scenario-permission-package-approval"
assert_target_exists "scenario-admin-tenant-boundary"
assert_target_depends_on "release-check" "scenario-admin-tenant-boundary"
assert_target_exists "scenario-admin-access-management"
assert_target_depends_on "release-check" "scenario-admin-access-management"
assert_target_exists "scenario-tenant-permission-center"
assert_target_depends_on "release-check" "scenario-tenant-permission-center"
assert_target_exists "evaluation-readiness"

assert_file_contains "Makefile" "scripts/scenario-tenant-permission-center.sh"
assert_file_contains "Makefile" "PNPM ?= ./scripts/pnpm.sh"
assert_file_contains "Makefile" '$(PNPM) --dir frontend install --frozen-lockfile'
assert_file_contains "Makefile" '$(PNPM) --dir scripts/real-mcp start'
assert_file_contains "Makefile" "scripts/pnpm.sh"
assert_file_contains "Makefile" "demo: scripts/demo.sh"
assert_file_contains "Makefile" "evaluation-readiness: scripts/evaluation-readiness.sh"
assert_file_contains "Makefile" "scripts/lib/ports.sh"
assert_file_contains "Makefile" "assert_port_free \"API\""
assert_file_contains "Makefile" "assert_port_free \"MCP\""
assert_file_contains "frontend/package.json" '"node": ">=24 <27"'
assert_file_contains ".github/workflows/ci.yml" "node-version: [24, 26]"
assert_file_contains ".github/workflows/ci.yml" 'node-version: ${{ matrix.node-version }}'
assert_file_contains ".github/workflows/ci.yml" "frontend-required:"
assert_file_contains ".github/workflows/ci.yml" "needs: frontend"
assert_file_contains ".github/workflows/ci.yml" "Frontend matrix passed"
assert_file_contains "scripts/scenario-web-console-production-journey.sh" "authRequired"
assert_file_contains "scripts/scenario-web-console-production-journey.sh" "productionAcceptance.ts"
assert_file_contains "scripts/scenario-web-console-production-journey.sh" "buildProductionAcceptanceCenter"
assert_file_contains "scripts/scenario-web-console-production-journey.sh" "connectionDiagnostics.ts"
assert_file_contains "scripts/scenario-web-console-production-journey.sh" "connection-diagnostics-action"
assert_file_contains "scripts/scenario-web-console-production-journey.sh" "tests/connectionDiagnostics.test.mjs"
assert_file_contains "scripts/scenario-web-console-production-journey.sh" "tests/productionAcceptance.test.mjs"
assert_file_contains "scripts/scenario-admin-tenant-boundary.sh" "AGENT_HARBOR_ADMIN_IDENTITIES"
assert_file_contains "scripts/scenario-admin-tenant-boundary.sh" "tenant_admin"
assert_file_contains "scripts/scenario-admin-tenant-boundary.sh" "403"
assert_file_contains "scripts/scenario-admin-access-management.sh" "/api/v1/admin-identities"
assert_file_contains "scripts/scenario-admin-access-management.sh" "key:rotate"
assert_file_contains "scripts/scenario-admin-access-management.sh" "admin_identity.disabled"
assert_file_contains "scripts/scenario-tenant-permission-center.sh" "/api/v1/tenants/"
assert_file_contains "scripts/scenario-tenant-permission-center.sh" "permission-center"
assert_file_contains "scripts/scenario-tenant-permission-center.sh" "operatorBoundary"
assert_file_contains "scripts/lib/ports.sh" "describe_port_owner"
assert_file_contains "scripts/lib/ports.sh" "assert_port_free"
assert_file_contains "scripts/scenario-production-hardening.sh" "scripts/lib/ports.sh"
assert_file_contains "scripts/scenario-production-hardening.sh" "assert_port_free"
assert_file_contains "scripts/pnpm.sh" 'PINNED_PNPM_VERSION="${AGENT_HARBOR_PNPM_VERSION:-10.30.3}"'
assert_file_contains "scripts/pnpm.sh" 'AGENT_HARBOR_PNPM_BIN'
assert_file_contains "scripts/demo.sh" 'PNPM:-$ROOT_DIR/scripts/pnpm.sh'
assert_file_contains "scripts/scenario-ai-admin-browser-journey.sh" "scripts/lib/ports.sh"
assert_file_contains "scripts/scenario-ai-admin-browser-journey.sh" "assert_port_free"
assert_file_contains "scripts/scenario-ai-admin-browser-journey.sh" 'PNPM:-$ROOT_DIR/scripts/pnpm.sh'
assert_file_contains "scripts/scenario-web-console-production-journey.sh" "scripts/lib/ports.sh"
assert_file_contains "scripts/scenario-web-console-production-journey.sh" "assert_port_free"
assert_file_contains "scripts/scenario-web-console-production-journey.sh" 'PNPM:-$ROOT_DIR/scripts/pnpm.sh'
assert_file_contains "scripts/scenario-admin-tenant-boundary.sh" "scripts/lib/ports.sh"
assert_file_contains "scripts/scenario-admin-tenant-boundary.sh" "assert_port_free"
assert_file_contains "scripts/scenario-admin-access-management.sh" "scripts/lib/ports.sh"
assert_file_contains "scripts/scenario-admin-access-management.sh" "assert_port_free"
assert_file_contains "scripts/scenario-tenant-permission-center.sh" "scripts/lib/ports.sh"
assert_file_contains "scripts/scenario-tenant-permission-center.sh" "assert_port_free"
assert_file_contains "scripts/scenario-permission-package-approval.sh" 'PNPM:-scripts/pnpm.sh'
assert_file_contains "scripts/demo.sh" "scripts/lib/ports.sh"
assert_file_contains "scripts/evaluation-readiness.sh" "feedback-log.csv"
assert_file_contains "scripts/evaluation-readiness.sh" "acceptance-report-notes.md"
assert_file_contains "README.md" "make evaluation-readiness"
assert_file_contains "docs/product/evaluation-readiness.md" "time-to-first-report"
assert_file_contains "docs/product/evaluation-readiness.md" "30-minute evaluator walkthrough"
