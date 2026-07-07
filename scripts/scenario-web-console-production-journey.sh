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
CLEANED_UP=0

# shellcheck source=scripts/lib/ports.sh
source "$ROOT_DIR/scripts/lib/ports.sh"

track_pid() {
  PIDS+=("$1")
  disown "$1" >/dev/null 2>&1 || true
}

cleanup() {
  if [[ "$CLEANED_UP" == "1" ]]; then
    return
  fi
  CLEANED_UP=1

  local pid
  for pid in "${PIDS[@]:-}"; do
    kill "$pid" >/dev/null 2>&1 || true
  done
  kill_port_listener "$API_PORT" TERM
  kill_port_listener "$MCP_PORT" TERM
  kill_port_listener "$FRONTEND_PORT" TERM
  sleep 0.2
  for pid in "${PIDS[@]:-}"; do
    if kill -0 "$pid" >/dev/null 2>&1; then
      kill -KILL "$pid" >/dev/null 2>&1 || true
    fi
  done
  kill_port_listener "$API_PORT" KILL
  kill_port_listener "$MCP_PORT" KILL
  kill_port_listener "$FRONTEND_PORT" KILL
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
track_pid "$!"

(cd scripts/real-mcp && REAL_MCP_HOST="$MCP_HOST" REAL_MCP_PORT="$MCP_PORT" node server.mjs) > "$LOG_DIR/mcp.log" 2>&1 &
track_pid "$!"

VITE_API_BASE="$BASE_URL" pnpm --dir frontend dev --host "$FRONTEND_HOST" --port "$FRONTEND_PORT" --strictPort > "$LOG_DIR/frontend.log" 2>&1 &
track_pid "$!"

wait_http "API" "$BASE_URL/healthz"
wait_http "MCP" "http://${MCP_HOST}:${MCP_PORT}/healthz"
wait_http "Web console" "$FRONTEND_ORIGIN/"

root_html="$(curl -fsS "$FRONTEND_ORIGIN/")"
assert_contains "web console root" 'id="root"' "$root_html"

system_info="$(curl -fsS "$BASE_URL/api/v1/system/info")"
python3 - "$system_info" <<'PY'
import json
import sys

payload = json.loads(sys.argv[1])
data = payload.get("data", {})
if data.get("authRequired") is not True:
    raise SystemExit(f"expected authRequired=true for production journey smoke gate, got {data.get('authRequired')!r}")
PY

production_journey_source="$(curl -fsS "$FRONTEND_ORIGIN/src/productionJourney.ts")"
production_acceptance_source="$(curl -fsS "$FRONTEND_ORIGIN/src/productionAcceptance.ts")"
checkpoint_source="$(curl -fsS "$FRONTEND_ORIGIN/src/components/ProductionJourneyCheckpoint.tsx")"
go_live_acceptance_source="$(curl -fsS "$FRONTEND_ORIGIN/src/components/GoLiveAcceptanceOverview.tsx")"
connection_diagnostics_source="$(curl -fsS "$FRONTEND_ORIGIN/src/connectionDiagnostics.ts")"
console_controller_source="$(curl -fsS "$FRONTEND_ORIGIN/src/ConsoleController.tsx")"
assert_contains "production journey model" "productionJourneyStages" "$production_journey_source"
assert_contains "production acceptance model" "buildProductionAcceptanceCenter" "$production_acceptance_source"
assert_contains "production journey checkpoint" "production-journey-checkpoint" "$checkpoint_source"
assert_contains "go-live acceptance model wiring" "buildProductionAcceptanceCenter" "$go_live_acceptance_source"
assert_contains "connection diagnostics model" "buildConnectionDiagnosticRows" "$connection_diagnostics_source"
assert_contains "connection diagnostics UI action" "connection-diagnostics-action" "$console_controller_source"
assert_contains "connection diagnostics UI list" "connection-diagnostics-list" "$console_controller_source"

for hash in getting-started registry ask ai-admin go-live; do
  curl -fsS "$FRONTEND_ORIGIN/#$hash" >/dev/null
  echo "route smoke ready: #$hash"
done

pnpm --dir frontend exec node --test \
  tests/connectionDiagnostics.test.mjs \
  tests/productionAcceptance.test.mjs \
  tests/productionJourney.test.mjs \
  tests/productionLanguage.test.mjs \
  tests/consoleNavigation.test.mjs

echo "Web console production journey smoke complete"
cleanup
trap - EXIT
