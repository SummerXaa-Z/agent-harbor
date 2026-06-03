#!/usr/bin/env bash
set -euo pipefail

API_HOST="${AGENT_HARBOR_DEMO_API_HOST:-127.0.0.1}"
API_PORT="${AGENT_HARBOR_DEMO_API_PORT:-9090}"
API_ADDR="${AGENT_HARBOR_ADDR:-${API_HOST}:${API_PORT}}"
FRONTEND_HOST="${AGENT_HARBOR_DEMO_FRONTEND_HOST:-127.0.0.1}"
FRONTEND_PORT="${AGENT_HARBOR_DEMO_FRONTEND_PORT:-5174}"
MOCK_MCP_HOST="${MOCK_MCP_HOST:-127.0.0.1}"
MOCK_MCP_PORT="${MOCK_MCP_PORT:-8787}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

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
need python3
need pnpm

assert_port_free "API" "$API_PORT"
assert_port_free "mock MCP" "$MOCK_MCP_PORT"
assert_port_free "frontend" "$FRONTEND_PORT"

cd "$ROOT_DIR"

echo "Starting AgentHarbor demo..."
echo "API:       http://${API_URL_HOST}:${API_PORT}"
echo "mock MCP:  http://${MOCK_MCP_HOST}:${MOCK_MCP_PORT}/mcp"
echo "console:   http://${FRONTEND_HOST}:${FRONTEND_PORT}"
echo

AGENT_HARBOR_ADDR="$API_ADDR" AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true go run ./cmd/agent-harbor &
PIDS+=("$!")

scripts/mock-mcp-server.py --host "$MOCK_MCP_HOST" --port "$MOCK_MCP_PORT" &
PIDS+=("$!")

pnpm --dir frontend dev --host "$FRONTEND_HOST" --port "$FRONTEND_PORT" --strictPort &
PIDS+=("$!")

echo "Demo is starting. Open http://${FRONTEND_HOST}:${FRONTEND_PORT}"
echo "Press Ctrl+C to stop all demo services."

supervise
