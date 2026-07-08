#!/usr/bin/env bash
set -euo pipefail

API_HOST="${AGENT_HARBOR_BROWSER_GATE_API_HOST:-127.0.0.1}"
API_PORT="${AGENT_HARBOR_BROWSER_GATE_API_PORT:-9198}"
API_ADDR="${AGENT_HARBOR_ADDR:-${API_HOST}:${API_PORT}}"
BASE_URL="${BASE_URL:-http://${API_HOST}:${API_PORT}}"
FRONTEND_HOST="${AGENT_HARBOR_BROWSER_GATE_FRONTEND_HOST:-127.0.0.1}"
FRONTEND_PORT="${AGENT_HARBOR_BROWSER_GATE_FRONTEND_PORT:-5184}"
FRONTEND_ORIGIN="${FRONTEND_ORIGIN:-http://${FRONTEND_HOST}:${FRONTEND_PORT}}"
MOCK_MCP_HOST="${MOCK_MCP_HOST:-127.0.0.1}"
MOCK_MCP_PORT="${MOCK_MCP_PORT:-8798}"
MCP_SERVER_MODE="${AGENT_HARBOR_BROWSER_GATE_MCP_MODE:-real}"
REQUESTER_ACTOR="${AGENT_HARBOR_BROWSER_GATE_REQUESTER_ACTOR:-requester}"
REVIEWER_ACTOR="${AGENT_HARBOR_BROWSER_GATE_REVIEWER_ACTOR:-security-reviewer}"
REQUESTER_ADMIN_KEY="${AGENT_HARBOR_BROWSER_GATE_REQUESTER_ADMIN_KEY:-browser-gate-requester-key}"
REVIEWER_ADMIN_KEY="${AGENT_HARBOR_BROWSER_GATE_REVIEWER_ADMIN_KEY:-browser-gate-reviewer-key}"
ADMIN_IDENTITIES="${AGENT_HARBOR_ADMIN_IDENTITIES:-${REQUESTER_ACTOR}=${REQUESTER_ADMIN_KEY};${REVIEWER_ACTOR}=${REVIEWER_ADMIN_KEY}}"
RUN_ID="${RUN_ID:-ai-admin-browser-journey-$(date +%Y%m%d%H%M%S)}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOG_DIR="${TMPDIR:-/tmp}/agent-harbor-browser-gate-${RUN_ID}"
PIDS=()
read -r -a PNPM_CMD <<< "${PNPM:-corepack pnpm}"

# shellcheck source=scripts/lib/ports.sh
source "$ROOT_DIR/scripts/lib/ports.sh"

cleanup() {
  local pid
  for pid in "${PIDS[@]:-}"; do
    kill "$pid" >/dev/null 2>&1 || true
  done
  sleep 0.5
  for pid in "${PIDS[@]:-}"; do
    if kill -0 "$pid" >/dev/null 2>&1; then
      kill -KILL "$pid" >/dev/null 2>&1 || true
    fi
  done
  for pid in "${PIDS[@]:-}"; do
    wait "$pid" >/dev/null 2>&1 || true
  done
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

verify_subject_header_cors() {
  local headers_file
  local status
  headers_file="$(mktemp)"
  status="$(
    curl -sS -o /dev/null -D "$headers_file" -w "%{http_code}" -X OPTIONS \
      "$BASE_URL/api/v1/mcp/agents/browser-gate/rpc" \
      -H "Origin: $FRONTEND_ORIGIN" \
      -H "Access-Control-Request-Method: POST" \
      -H "Access-Control-Request-Headers: Authorization, Content-Type, X-Admin-Key, X-Run-Id, X-AgentHarbor-Subject-Id"
  )"
  if [[ "$status" != "204" ]]; then
    echo "expected CORS preflight status 204, got $status" >&2
    cat "$headers_file" >&2
    rm -f "$headers_file"
    exit 1
  fi
  if ! tr -d '\r' < "$headers_file" | grep -i '^Access-Control-Allow-Headers:' | grep -qi 'X-AgentHarbor-Subject-Id'; then
    echo "CORS preflight did not allow X-AgentHarbor-Subject-Id" >&2
    cat "$headers_file" >&2
    rm -f "$headers_file"
    exit 1
  fi
  rm -f "$headers_file"
  echo "browser subject-header CORS preflight verified"
}

need curl
need go
need node
need "${PNPM_CMD[0]}"
need python3

assert_port_free "API" "$API_PORT"
assert_port_free "MCP" "$MOCK_MCP_PORT"
assert_port_free "frontend" "$FRONTEND_PORT"

mkdir -p "$LOG_DIR"
cd "$ROOT_DIR"

echo "AgentHarbor AI Admin browser journey gate"
echo "BASE_URL=$BASE_URL"
echo "FRONTEND_ORIGIN=$FRONTEND_ORIGIN"
echo "MCP=http://${MOCK_MCP_HOST}:${MOCK_MCP_PORT}/mcp (${MCP_SERVER_MODE})"
echo "ADMIN_IDENTITIES=${REQUESTER_ACTOR}/${REVIEWER_ACTOR}"
echo "RUN_ID=$RUN_ID"

AGENT_HARBOR_ADDR="$API_ADDR" AGENT_HARBOR_ADMIN_IDENTITIES="$ADMIN_IDENTITIES" AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true AGENT_HARBOR_CORS_ORIGINS="$FRONTEND_ORIGIN" go run ./cmd/agent-harbor > "$LOG_DIR/api.log" 2>&1 &
PIDS+=("$!")

case "$MCP_SERVER_MODE" in
  real)
    "${PNPM_CMD[@]}" --dir scripts/real-mcp install --frozen-lockfile >/dev/null
    (cd scripts/real-mcp && REAL_MCP_HOST="$MOCK_MCP_HOST" REAL_MCP_PORT="$MOCK_MCP_PORT" node server.mjs) > "$LOG_DIR/mcp.log" 2>&1 &
    PIDS+=("$!")
    ;;
  mock)
    scripts/mock-mcp-server.py --host "$MOCK_MCP_HOST" --port "$MOCK_MCP_PORT" > "$LOG_DIR/mcp.log" 2>&1 &
    PIDS+=("$!")
    ;;
  *)
    echo "AGENT_HARBOR_BROWSER_GATE_MCP_MODE must be real or mock" >&2
    exit 1
    ;;
esac

VITE_API_BASE="$BASE_URL" "${PNPM_CMD[@]}" --dir frontend dev --host "$FRONTEND_HOST" --port "$FRONTEND_PORT" --strictPort > "$LOG_DIR/frontend.log" 2>&1 &
PIDS+=("$!")

wait_http "API" "$BASE_URL/healthz"
wait_http "MCP" "http://${MOCK_MCP_HOST}:${MOCK_MCP_PORT}/healthz"
wait_http "Web console" "$FRONTEND_ORIGIN/"

if ! curl -fsS "$FRONTEND_ORIGIN/" | grep -q 'id="root"'; then
  echo "web console root element was not served" >&2
  show_logs
  exit 1
fi

verify_subject_header_cors

START_MOCK_MCP=false \
  BASE_URL="$BASE_URL" \
  REQUESTER_ACTOR="$REQUESTER_ACTOR" \
  APPROVAL_REVIEWER="$REVIEWER_ACTOR" \
  REQUESTER_ADMIN_KEY="$REQUESTER_ADMIN_KEY" \
  REVIEWER_ADMIN_KEY="$REVIEWER_ADMIN_KEY" \
  ADMIN_KEY="$REQUESTER_ADMIN_KEY" \
  RUN_ID="$RUN_ID" \
  MCP_SERVER_MODE="$MCP_SERVER_MODE" \
  MOCK_MCP_HOST="$MOCK_MCP_HOST" \
  MOCK_MCP_PORT="$MOCK_MCP_PORT" \
  bash scripts/scenario-permission-package-approval.sh

echo "AI Admin browser journey gate complete"
