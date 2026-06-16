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
assert_target_depends_on "demo" "frontend-deps"
assert_target_depends_on "ai-admin-browser-journey" "frontend-deps"
assert_target_depends_on "release-check" "production-hardening"
assert_target_depends_on "web-console-production-journey" "frontend-deps"
assert_target_depends_on "web-console-production-journey" "real-mcp-deps"
assert_target_depends_on "release-check" "web-console-production-journey"
assert_target_exists "scenario-admin-tenant-boundary"
assert_target_depends_on "release-check" "scenario-admin-tenant-boundary"
assert_target_exists "scenario-admin-access-management"
assert_target_depends_on "release-check" "scenario-admin-access-management"
assert_target_exists "scenario-tenant-permission-center"
assert_target_depends_on "release-check" "scenario-tenant-permission-center"

assert_file_contains "Makefile" "scripts/scenario-tenant-permission-center.sh"
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
