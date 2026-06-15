# Production Acceptance Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the known frontend dependency alerts and add a dependency-free production web-console smoke gate for the main operator journey.

**Architecture:** Use a pnpm override to force Vite's transitive `esbuild` dependency to the patched `0.28.1` release, with lockfile regression tests. Add a shell smoke script that starts isolated local services, verifies the console is served, and runs the existing production journey/language/navigation tests as a release gate without adding browser automation dependencies.

**Tech Stack:** pnpm 10, Vite 8, Node `node:test`, Bash, Go, existing AgentHarbor scenario scripts.

---

## File Structure

- Modify `frontend/package.json`
  - Add a `pnpm.overrides.esbuild` pin to `0.28.1`.
- Modify `frontend/pnpm-lock.yaml`
  - Resolve Vite's accepted transitive `esbuild` range to `0.28.1`.
- Modify `frontend/tests/viteConfig.test.mjs`
  - Add security-regression assertions for the override and lockfile.
- Create `scripts/scenario-web-console-production-journey.sh`
  - Start API, real MCP, and Vite on isolated ports; verify health and production journey smoke signals.
- Modify `Makefile`
  - Add `web-console-production-journey` target, include it in `release-check`, and lint it with scenario scripts.
- Modify `tests/makefile_targets_test.sh`
  - Assert target dependencies.
- Modify `README.md`, `docs/engineering/release-checklist.md`, and `CHANGELOG.md`
  - Document the new gate and dependency security closure.
- Modify this plan while executing
  - Check off each step after verification.

---

### Task 1: Patch Frontend Dependency Security

**Files:**
- Modify: `frontend/package.json`
- Modify: `frontend/pnpm-lock.yaml`
- Modify: `frontend/tests/viteConfig.test.mjs`

- [ ] **Step 1: Add failing dependency-security assertions**

Extend `frontend/tests/viteConfig.test.mjs`:

```js
const packageJson = JSON.parse(readFileSync(new URL("../package.json", import.meta.url), "utf8"));
const lockfile = readFileSync(new URL("../pnpm-lock.yaml", import.meta.url), "utf8");
```

Add:

```js
test("vite esbuild transitive dependency is pinned to the patched line", () => {
  assert.equal(packageJson.pnpm?.overrides?.esbuild, "0.28.1");
  assert.doesNotMatch(lockfile, /esbuild@0\.27\.7/);
  assert.match(lockfile, /esbuild@0\.28\.1/);
});
```

- [ ] **Step 2: Run the focused test and confirm RED**

Run:

```bash
pnpm --dir frontend exec node --test tests/viteConfig.test.mjs
```

Expected: FAIL because `packageJson.pnpm.overrides.esbuild` is missing and the lockfile still resolves `esbuild@0.27.7`.

- [ ] **Step 3: Add the pnpm override**

Modify `frontend/package.json` so the root object includes:

```json
"pnpm": {
  "overrides": {
    "esbuild": "0.28.1"
  }
}
```

Keep existing dependency versions unchanged.

- [ ] **Step 4: Refresh the lockfile**

Run:

```bash
pnpm --dir frontend install --lockfile-only
```

Expected: `frontend/pnpm-lock.yaml` resolves `esbuild@0.28.1` and its platform packages to `0.28.1`.

- [ ] **Step 5: Run focused dependency verification**

Run:

```bash
pnpm --dir frontend why esbuild
pnpm --dir frontend exec node --test tests/viteConfig.test.mjs
```

Expected: `pnpm why esbuild` reports `esbuild@0.28.1`; focused test passes.

---

### Task 2: Add Web Console Production Journey Smoke Gate

**Files:**
- Create: `scripts/scenario-web-console-production-journey.sh`
- Modify: `Makefile`
- Modify: `tests/makefile_targets_test.sh`

- [ ] **Step 1: Add failing Makefile target assertions**

Modify `tests/makefile_targets_test.sh`:

```bash
assert_target_depends_on "web-console-production-journey" "frontend-deps"
assert_target_depends_on "web-console-production-journey" "real-mcp-deps"
assert_target_depends_on "release-check" "web-console-production-journey"
```

- [ ] **Step 2: Run Makefile target test and confirm RED**

Run:

```bash
bash tests/makefile_targets_test.sh
```

Expected: FAIL because `web-console-production-journey` does not exist.

- [ ] **Step 3: Create the smoke script**

Create `scripts/scenario-web-console-production-journey.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

API_HOST="${AGENT_HARBOR_WEB_GATE_API_HOST:-127.0.0.1}"
API_PORT="${AGENT_HARBOR_WEB_GATE_API_PORT:-9193}"
API_ADDR="${API_HOST}:${API_PORT}"
BASE_URL="http://${API_HOST}:${API_PORT}"
FRONTEND_HOST="${AGENT_HARBOR_WEB_GATE_FRONTEND_HOST:-127.0.0.1}"
FRONTEND_PORT="${AGENT_HARBOR_WEB_GATE_FRONTEND_PORT:-5183}"
FRONTEND_ORIGIN="http://${FRONTEND_HOST}:${FRONTEND_PORT}"
MCP_HOST="${AGENT_HARBOR_WEB_GATE_MCP_HOST:-127.0.0.1}"
MCP_PORT="${AGENT_HARBOR_WEB_GATE_MCP_PORT:-8793}"
RUN_ID="${RUN_ID:-web-console-production-journey-$(date +%Y%m%d%H%M%S)}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOG_DIR="${TMPDIR:-/tmp}/agent-harbor-web-gate-${RUN_ID}"
PIDS=()

cleanup() {
  local pid
  for pid in "${PIDS[@]:-}"; do
    kill "$pid" >/dev/null 2>&1 || true
  done
  wait >/dev/null 2>&1 || true
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing dependency: $1" >&2
    exit 1
  fi
}

port_in_use() {
  local port="$1"
  if command -v lsof >/dev/null 2>&1; then
    lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1
    return
  fi
  python3 - "$port" <<'PY'
import socket
import sys

port = int(sys.argv[1])
sock = socket.socket()
try:
    sock.bind(("127.0.0.1", port))
except OSError:
    raise SystemExit(0)
finally:
    sock.close()
raise SystemExit(1)
PY
}

assert_port_free() {
  local label="$1"
  local port="$2"
  if port_in_use "$port"; then
    echo "$label port $port is already in use" >&2
    exit 1
  fi
}

show_logs() {
  local file
  for file in "$LOG_DIR"/*.log; do
    [[ -f "$file" ]] || continue
    echo "== $file ==" >&2
    tail -80 "$file" >&2 || true
  done
}

wait_http() {
  local label="$1"
  local url="$2"
  local i
  for i in $(seq 1 60); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      echo "$label ready: $url"
      return
    fi
    sleep 0.5
  done
  echo "$label did not become ready: $url" >&2
  show_logs
  exit 1
}

assert_contains() {
  local label="$1"
  local needle="$2"
  local haystack="$3"
  if [[ "$haystack" != *"$needle"* ]]; then
    echo "$label missing expected text: $needle" >&2
    exit 1
  fi
}

need curl
need go
need node
need pnpm
need python3

assert_port_free "API" "$API_PORT"
assert_port_free "MCP" "$MCP_PORT"
assert_port_free "frontend" "$FRONTEND_PORT"

mkdir -p "$LOG_DIR"
cd "$ROOT_DIR"

echo "AgentHarbor web console production journey smoke"
echo "BASE_URL=$BASE_URL"
echo "FRONTEND_ORIGIN=$FRONTEND_ORIGIN"
echo "MCP=http://${MCP_HOST}:${MCP_PORT}/mcp"
echo "RUN_ID=$RUN_ID"

AGENT_HARBOR_ADDR="$API_ADDR" AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true go run ./cmd/agent-harbor > "$LOG_DIR/api.log" 2>&1 &
PIDS+=("$!")

(cd scripts/real-mcp && REAL_MCP_HOST="$MCP_HOST" REAL_MCP_PORT="$MCP_PORT" node server.mjs) > "$LOG_DIR/mcp.log" 2>&1 &
PIDS+=("$!")

VITE_API_BASE="$BASE_URL" pnpm --dir frontend dev --host "$FRONTEND_HOST" --port "$FRONTEND_PORT" --strictPort > "$LOG_DIR/frontend.log" 2>&1 &
PIDS+=("$!")

wait_http "API" "$BASE_URL/healthz"
wait_http "MCP" "http://${MCP_HOST}:${MCP_PORT}/healthz"
wait_http "Web console" "$FRONTEND_ORIGIN/"

root_html="$(curl -fsS "$FRONTEND_ORIGIN/")"
assert_contains "web console root" 'id="root"' "$root_html"

production_journey_source="$(curl -fsS "$FRONTEND_ORIGIN/src/productionJourney.ts")"
checkpoint_source="$(curl -fsS "$FRONTEND_ORIGIN/src/components/ProductionJourneyCheckpoint.tsx")"
assert_contains "production journey model" "productionJourneyStages" "$production_journey_source"
assert_contains "production journey checkpoint" "production-journey-checkpoint" "$checkpoint_source"

for hash in getting-started registry ask ai-admin evidence; do
  curl -fsS "$FRONTEND_ORIGIN/#$hash" >/dev/null
  echo "route smoke ready: #$hash"
done

pnpm --dir frontend exec node --test \
  tests/productionJourney.test.mjs \
  tests/productionLanguage.test.mjs \
  tests/consoleNavigation.test.mjs

echo "Web console production journey smoke complete"
```

- [ ] **Step 4: Wire the Makefile target**

Modify `Makefile`:

- Add `scripts/scenario-web-console-production-journey.sh` to `SCENARIO_SCRIPTS`.
- Add `web-console-production-journey` to `.PHONY`.
- Add help text:

```make
	@printf '  make web-console-production-journey Run the web console production journey smoke gate\n'
```

- Add `web-console-production-journey` to `release-check` after `production-hardening`.
- Add target:

```make
web-console-production-journey: frontend-deps real-mcp-deps
	bash scripts/scenario-web-console-production-journey.sh
```

- [ ] **Step 5: Run focused smoke verification**

Run:

```bash
bash tests/makefile_targets_test.sh
bash -n scripts/scenario-web-console-production-journey.sh
AGENT_HARBOR_WEB_GATE_API_PORT=9194 AGENT_HARBOR_WEB_GATE_MCP_PORT=8794 AGENT_HARBOR_WEB_GATE_FRONTEND_PORT=5184 bash scripts/scenario-web-console-production-journey.sh
```

Expected: all pass and the smoke script prints `Web console production journey smoke complete`.

---

### Task 3: Documentation and Changelog

**Files:**
- Modify: `README.md`
- Modify: `docs/engineering/release-checklist.md`
- Modify: `CHANGELOG.md`
- Modify: this plan

- [ ] **Step 1: Update README**

Document:

```md
make web-console-production-journey
```

Explain that this starts an isolated local API, real MCP demo, and web console, then verifies the production journey smoke signals without adding browser automation dependencies.

- [ ] **Step 2: Update release checklist**

Add `make web-console-production-journey` next to `make production-hardening` and clarify that `make release-check` includes both.

- [ ] **Step 3: Update CHANGELOG**

Add under `## [Unreleased]`:

```md
- Frontend dependency security now pins Vite's transitive `esbuild` resolution to the patched `0.28.1` line through a pnpm override, with tests guarding against lockfile downgrade.
- 前端依赖安全现在通过 pnpm override 将 Vite 传递依赖 `esbuild` 固定到已修复的 `0.28.1` 版本，并用测试防止 lockfile 回退。
- Release readiness now includes `make web-console-production-journey`, a dependency-free smoke gate for the served production journey console path.
- 发布验收现在包含 `make web-console-production-journey`，用于对已启动的生产旅程控制台路径执行无新增依赖的 smoke gate。
```

- [ ] **Step 4: Run docs and focused verification**

Run:

```bash
pnpm --dir frontend exec node --test tests/viteConfig.test.mjs
bash tests/makefile_targets_test.sh
git diff --check
```

Expected: all pass.

---

### Task 4: Full Gates, Commit, and PR

**Files:**
- Modify: this plan

- [ ] **Step 1: Run full gates**

Run:

```bash
pnpm --dir frontend test
pnpm --dir frontend build
make check
make release-check
```

Expected: all pass.

- [ ] **Step 2: Check Dependabot alert state**

Run:

```bash
gh api repos/SummerXaa-Z/agent-harbor/dependabot/alerts --jq '.[] | select(.dependency.package.name=="esbuild") | {number,state,severity: .security_advisory.severity, package: .dependency.package.name, manifest: .dependency.manifest_path, vulnerable: .security_vulnerability.vulnerable_version_range, patched: .security_vulnerability.first_patched_version.identifier}'
```

Expected: before merge, default-branch alerts may still show open; PR description should state the branch lockfile resolves `esbuild@0.28.1`.

- [ ] **Step 3: Commit**

Run:

```bash
git add CHANGELOG.md README.md docs/engineering/release-checklist.md docs/superpowers/plans/2026-06-15-production-acceptance-hardening.md frontend/package.json frontend/pnpm-lock.yaml frontend/tests/viteConfig.test.mjs Makefile tests/makefile_targets_test.sh scripts/scenario-web-console-production-journey.sh
git commit -m "chore: harden production acceptance gates"
```

- [ ] **Step 4: Push and create PR**

Run:

```bash
git push -u origin codex/production-acceptance-hardening
gh pr create --base main --head codex/production-acceptance-hardening --title "Harden production acceptance gates" --body "<summary and verification>"
```

Expected: PR opens against `main`.

---

## Self-Review

- Spec coverage: dependency alert closure, smoke gate, Makefile integration, docs, focused gates, full gates, and PR evidence are covered.
- Placeholder scan: no placeholder steps remain.
- Boundary check: no backend permission logic changes and no new npm dependencies are planned.
