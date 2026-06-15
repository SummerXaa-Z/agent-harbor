# Connection Diagnostics Smoke Gate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the web-console production journey smoke gate so release readiness verifies the connection diagnostics contract from the served local stack.

**Architecture:** Keep the gate dependency-free. `scripts/scenario-web-console-production-journey.sh` will query the running production-shaped API and served source files with `curl`, parse JSON with `python3`, and include the focused connection diagnostics unit test in the route-level test command.

**Tech Stack:** Bash, curl, python3, existing Node test runner, Make release gates.

---

### Task 1: Source Guard Red Test

**Files:**
- Modify: `tests/makefile_targets_test.sh`

- [x] **Step 1: Add failing guard assertions**

Add helpers:

```bash
assert_file_contains() {
  local file="$1"
  local needle="$2"
  if ! grep -Fq "$needle" "$file"; then
    echo "expected ${file} to contain ${needle}" >&2
    exit 1
  fi
}
```

Then assert:

```bash
assert_file_contains "scripts/scenario-web-console-production-journey.sh" "authRequired"
assert_file_contains "scripts/scenario-web-console-production-journey.sh" "connectionDiagnostics.ts"
assert_file_contains "scripts/scenario-web-console-production-journey.sh" "connection-diagnostics-action"
assert_file_contains "scripts/scenario-web-console-production-journey.sh" "tests/connectionDiagnostics.test.mjs"
```

- [x] **Step 2: Verify guard fails**

Run:

```bash
bash tests/makefile_targets_test.sh
```

Expected: FAIL because the scenario script does not yet contain the connection diagnostics smoke checks.

### Task 2: Scenario Smoke Implementation

**Files:**
- Modify: `scripts/scenario-web-console-production-journey.sh`

- [x] **Step 1: Add system info auth metadata check**

After root HTML assertions, add:

```bash
system_info="$(curl -fsS "$BASE_URL/api/v1/system/info")"
python3 - "$system_info" <<'PY'
import json
import sys

payload = json.loads(sys.argv[1])
data = payload.get("data", {})
if data.get("authRequired") is not True:
    raise SystemExit(f"expected authRequired=true for production journey smoke gate, got {data.get('authRequired')!r}")
PY
```

- [x] **Step 2: Add served source assertions**

Fetch and assert:

```bash
connection_diagnostics_source="$(curl -fsS "$FRONTEND_ORIGIN/src/connectionDiagnostics.ts")"
console_controller_source="$(curl -fsS "$FRONTEND_ORIGIN/src/ConsoleController.tsx")"
assert_contains "connection diagnostics model" "buildConnectionDiagnosticRows" "$connection_diagnostics_source"
assert_contains "connection diagnostics UI action" "connection-diagnostics-action" "$console_controller_source"
assert_contains "connection diagnostics UI list" "connection-diagnostics-list" "$console_controller_source"
```

- [x] **Step 3: Include focused diagnostics test in scenario**

Extend the existing Node test command:

```bash
pnpm --dir frontend exec node --test \
  tests/connectionDiagnostics.test.mjs \
  tests/productionJourney.test.mjs \
  tests/productionLanguage.test.mjs \
  tests/consoleNavigation.test.mjs
```

- [x] **Step 4: Verify focused smoke passes**

Run:

```bash
bash tests/makefile_targets_test.sh
AGENT_HARBOR_WEB_GATE_API_PORT=9194 AGENT_HARBOR_WEB_GATE_MCP_PORT=8794 AGENT_HARBOR_WEB_GATE_FRONTEND_PORT=5184 bash scripts/scenario-web-console-production-journey.sh
```

Expected: both commands exit 0.

### Task 3: Docs and Release Gates

**Files:**
- Modify: `README.md`
- Modify: `CHANGELOG.md`

- [x] **Step 1: Update docs**

Mention that `make web-console-production-journey` also checks connection diagnostics and system-info auth metadata.

- [x] **Step 2: Update changelog**

Add English and Chinese bullets under Unreleased.

- [x] **Step 3: Run full verification**

Run:

```bash
pnpm --dir frontend test
pnpm --dir frontend build
make check
make release-check
git diff --check
```

Expected: all commands exit 0.
