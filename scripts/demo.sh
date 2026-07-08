#!/usr/bin/env bash
set -euo pipefail

API_HOST="${AGENT_HARBOR_DEMO_API_HOST:-127.0.0.1}"
API_PORT="${AGENT_HARBOR_DEMO_API_PORT:-9090}"
API_ADDR="${AGENT_HARBOR_ADDR:-${API_HOST}:${API_PORT}}"
FRONTEND_HOST="${AGENT_HARBOR_DEMO_FRONTEND_HOST:-127.0.0.1}"
FRONTEND_PORT="${AGENT_HARBOR_DEMO_FRONTEND_PORT:-5174}"
MOCK_MCP_HOST="${MOCK_MCP_HOST:-127.0.0.1}"
MOCK_MCP_PORT="${MOCK_MCP_PORT:-8787}"
MCP_SERVER_MODE="${AGENT_HARBOR_DEMO_MCP_MODE:-real}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
read -r -a PNPM_CMD <<< "${PNPM:-corepack pnpm}"

# shellcheck source=scripts/lib/ports.sh
source "$ROOT_DIR/scripts/lib/ports.sh"

PIDS=()

if [[ -n "${AGENT_HARBOR_ADDR:-}" ]]; then
  if [[ "$API_ADDR" =~ ^\[([^]]+)\]:([0-9]+)$ ]]; then
    API_HOST="${BASH_REMATCH[1]}"
    API_PORT="${BASH_REMATCH[2]}"
  elif [[ "$API_ADDR" =~ ^:([0-9]+)$ ]]; then
    API_HOST="127.0.0.1"
    API_PORT="${BASH_REMATCH[1]}"
  elif [[ "$API_ADDR" =~ ^([^:]+):([0-9]+)$ ]]; then
    API_HOST="${BASH_REMATCH[1]}"
    API_PORT="${BASH_REMATCH[2]}"
  else
    echo "AGENT_HARBOR_ADDR must include a numeric port, for example 127.0.0.1:9090" >&2
    exit 1
  fi
fi
API_URL_HOST="$API_HOST"
if [[ "$API_URL_HOST" == *:* && "$API_URL_HOST" != \[* ]]; then
  API_URL_HOST="[${API_URL_HOST}]"
fi
API_BASE_URL="http://${API_URL_HOST}:${API_PORT}"
FRONTEND_URL_HOST="$FRONTEND_HOST"
if [[ "$FRONTEND_URL_HOST" == *:* && "$FRONTEND_URL_HOST" != \[* ]]; then
  FRONTEND_URL_HOST="[${FRONTEND_URL_HOST}]"
fi
FRONTEND_ORIGIN="http://${FRONTEND_URL_HOST}:${FRONTEND_PORT}"

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

is_running() {
  local target="$1"
  local pid
  for pid in $(jobs -pr); do
    if [[ "$pid" == "$target" ]]; then
      return 0
    fi
  done
  return 1
}

supervise() {
  local pid
  local status
  while true; do
    for pid in "${PIDS[@]}"; do
      if ! is_running "$pid"; then
        set +e
        wait "$pid"
        status=$?
        set -e
        echo "demo service PID $pid exited with status $status" >&2
        exit 1
      fi
    done
    sleep 1
  done
}

need go
need node
need python3
need "${PNPM_CMD[0]}"

assert_port_free "API" "$API_PORT"
assert_port_free "MCP" "$MOCK_MCP_PORT"
assert_port_free "frontend" "$FRONTEND_PORT"

cd "$ROOT_DIR"

"${PNPM_CMD[@]}" --dir frontend install --frozen-lockfile

echo "Starting AgentHarbor demo..."
echo "API:       ${API_BASE_URL}"
echo "MCP:       http://${MOCK_MCP_HOST}:${MOCK_MCP_PORT}/mcp (${MCP_SERVER_MODE})"
echo "console:   ${FRONTEND_ORIGIN}"
echo

AGENT_HARBOR_ADDR="$API_ADDR" \
  AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true \
  AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true \
  AGENT_HARBOR_CORS_ORIGINS="${AGENT_HARBOR_CORS_ORIGINS:-$FRONTEND_ORIGIN}" \
  go run ./cmd/agent-harbor &
PIDS+=("$!")

case "$MCP_SERVER_MODE" in
  real)
    "${PNPM_CMD[@]}" --dir scripts/real-mcp install --frozen-lockfile >/dev/null
    (cd scripts/real-mcp && REAL_MCP_HOST="$MOCK_MCP_HOST" REAL_MCP_PORT="$MOCK_MCP_PORT" node server.mjs) &
    PIDS+=("$!")
    ;;
  mock)
    scripts/mock-mcp-server.py --host "$MOCK_MCP_HOST" --port "$MOCK_MCP_PORT" &
    PIDS+=("$!")
    ;;
  *)
    echo "AGENT_HARBOR_DEMO_MCP_MODE must be real or mock" >&2
    exit 1
    ;;
esac

VITE_API_BASE="${VITE_API_BASE:-$API_BASE_URL}" "${PNPM_CMD[@]}" --dir frontend dev --host "$FRONTEND_HOST" --port "$FRONTEND_PORT" --strictPort &
PIDS+=("$!")

echo "Demo is starting. Open ${FRONTEND_ORIGIN}"
echo "Press Ctrl+C to stop all demo services."

supervise
