#!/usr/bin/env bash
set -euo pipefail

API_HOST="${AGENT_HARBOR_BROWSER_GATE_API_HOST:-127.0.0.1}"
API_PORT="${AGENT_HARBOR_BROWSER_GATE_API_PORT:-9090}"
API_ADDR="${AGENT_HARBOR_ADDR:-${API_HOST}:${API_PORT}}"
BASE_URL="${BASE_URL:-http://${API_HOST}:${API_PORT}}"
FRONTEND_HOST="${AGENT_HARBOR_BROWSER_GATE_FRONTEND_HOST:-127.0.0.1}"
FRONTEND_PORT="${AGENT_HARBOR_BROWSER_GATE_FRONTEND_PORT:-5174}"
FRONTEND_ORIGIN="${FRONTEND_ORIGIN:-http://${FRONTEND_HOST}:${FRONTEND_PORT}}"
MOCK_MCP_HOST="${MOCK_MCP_HOST:-127.0.0.1}"
MOCK_MCP_PORT="${MOCK_MCP_PORT:-8787}"
RUN_ID="${RUN_ID:-ai-admin-browser-journey-$(date +%Y%m%d%H%M%S)}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOG_DIR="${TMPDIR:-/tmp}/agent-harbor-browser-gate-${RUN_ID}"
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
need pnpm
need python3

assert_port_free "API" "$API_PORT"
assert_port_free "mock MCP" "$MOCK_MCP_PORT"
assert_port_free "frontend" "$FRONTEND_PORT"

mkdir -p "$LOG_DIR"
cd "$ROOT_DIR"

echo "AgentHarbor AI Admin browser journey gate"
echo "BASE_URL=$BASE_URL"
echo "FRONTEND_ORIGIN=$FRONTEND_ORIGIN"
echo "MOCK_MCP=http://${MOCK_MCP_HOST}:${MOCK_MCP_PORT}/mcp"
echo "RUN_ID=$RUN_ID"

AGENT_HARBOR_ADDR="$API_ADDR" AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true go run ./cmd/agent-harbor > "$LOG_DIR/api.log" 2>&1 &
PIDS+=("$!")

scripts/mock-mcp-server.py --host "$MOCK_MCP_HOST" --port "$MOCK_MCP_PORT" > "$LOG_DIR/mock-mcp.log" 2>&1 &
PIDS+=("$!")

VITE_API_BASE="$BASE_URL" pnpm --dir frontend dev --host "$FRONTEND_HOST" --port "$FRONTEND_PORT" --strictPort > "$LOG_DIR/frontend.log" 2>&1 &
PIDS+=("$!")

wait_http "API" "$BASE_URL/healthz"
wait_http "Mock MCP" "http://${MOCK_MCP_HOST}:${MOCK_MCP_PORT}/healthz"
wait_http "Web console" "$FRONTEND_ORIGIN/"

if ! curl -fsS "$FRONTEND_ORIGIN/" | grep -q 'id="root"'; then
  echo "web console root element was not served" >&2
  show_logs
  exit 1
fi

verify_subject_header_cors

START_MOCK_MCP=false \
  BASE_URL="$BASE_URL" \
  RUN_ID="$RUN_ID" \
  MOCK_MCP_HOST="$MOCK_MCP_HOST" \
  MOCK_MCP_PORT="$MOCK_MCP_PORT" \
  bash scripts/scenario-permission-package-approval.sh

echo "AI Admin browser journey gate complete"
