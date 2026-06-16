#!/usr/bin/env bash
set -euo pipefail

API_HOST="${AGENT_HARBOR_ADMIN_BOUNDARY_API_HOST:-127.0.0.1}"
API_PORT="${AGENT_HARBOR_ADMIN_BOUNDARY_API_PORT:-9194}"
API_ADDR="${AGENT_HARBOR_ADMIN_BOUNDARY_API_ADDR:-${API_HOST}:${API_PORT}}"
BASE_URL="${AGENT_HARBOR_ADMIN_BOUNDARY_BASE_URL:-http://${API_HOST}:${API_PORT}}"
RUN_ID="${RUN_ID:-admin-boundary-$(date +%Y%m%d%H%M%S)}"
PLATFORM_KEY="${PLATFORM_KEY:-platform-key}"
EAST_KEY="${EAST_KEY:-east-key}"
ROOT_TENANT_ID="tenant-root"
EAST_TENANT_ID="tenant-east"
WEST_TENANT_ID="tenant-west"
EAST_WORKSPACE_ID="ws-support"
WEST_WORKSPACE_ID="ws-finance"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
LOG_DIR="${TMPDIR:-/tmp}/agent-harbor-admin-boundary-${RUN_ID}"
PIDS=()
HTTP_STATUS=""
HTTP_BODY=""

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

request() {
  local method="$1"
  local path="$2"
  local auth_mode="${3:-platform}"
  local body="${4:-}"
  local tmp
  tmp="$(mktemp)"

  local args=(
    -sS
    -o "$tmp"
    -w "%{http_code}"
    -X "$method"
    "$BASE_URL$path"
    -H "Content-Type: application/json"
  )

  case "$auth_mode" in
    platform)
      args+=(-H "X-Admin-Key: $PLATFORM_KEY")
      ;;
    east)
      args+=(-H "X-Admin-Key: $EAST_KEY")
      ;;
    none)
      ;;
    *)
      rm -f "$tmp"
      echo "unknown auth mode: $auth_mode" >&2
      exit 1
      ;;
  esac

  if [[ -n "$body" ]]; then
    args+=(-d "$body")
  fi

  if ! HTTP_STATUS="$(curl "${args[@]}")"; then
    rm -f "$tmp"
    echo "curl failed for $method $path" >&2
    show_logs
    exit 1
  fi
  HTTP_BODY="$(<"$tmp")"
  rm -f "$tmp"
}

expect_status() {
  local expected="$1"
  local label="$2"
  if [[ "$HTTP_STATUS" != "$expected" ]]; then
    echo "expected $label status $expected, got $HTTP_STATUS" >&2
    echo "$HTTP_BODY" >&2
    show_logs
    exit 1
  fi
}

json_body() {
  python3 - "$@" <<'PY'
import json
import sys

kind = sys.argv[1]
if kind == "tenant":
    tenant_id, parent_id, name = sys.argv[2], sys.argv[3], sys.argv[4]
    body = {"id": tenant_id, "name": name, "status": "active"}
    if parent_id:
        body["parentTenantId"] = parent_id
elif kind == "agent":
    tenant_id, workspace_id, name = sys.argv[2], sys.argv[3], sys.argv[4]
    body = {
        "tenantId": tenant_id,
        "workspaceId": workspace_id,
        "name": name,
        "channelType": "local",
        "status": "active",
    }
elif kind == "mcp-list-agents":
    arguments = {}
    if len(sys.argv) > 2 and sys.argv[2]:
        arguments["tenantId"] = sys.argv[2]
    if len(sys.argv) > 3 and sys.argv[3]:
        arguments["workspaceId"] = sys.argv[3]
    suffix = arguments.get("tenantId") or "scoped"
    body = {
        "jsonrpc": "2.0",
        "id": f"admin-boundary-list-agents-{suffix}",
        "method": "tools/call",
        "params": {"name": "list_agents", "arguments": arguments},
    }
else:
    raise SystemExit(f"unknown body kind: {kind}")
print(json.dumps(body, separators=(",", ":")))
PY
}

assert_agents_scope() {
  local expected_count="$1"
  local expected_tenant="$2"
  local expected_workspace="$3"
  RESPONSE_BODY="$HTTP_BODY" python3 - "$expected_count" "$expected_tenant" "$expected_workspace" <<'PY'
import json
import os
import sys

expected_count = int(sys.argv[1])
expected_tenant = sys.argv[2]
expected_workspace = sys.argv[3]
doc = json.loads(os.environ["RESPONSE_BODY"])
rows = doc["data"]
if len(rows) != expected_count:
    raise SystemExit(f"expected {expected_count} agent rows, got {len(rows)}: {rows}")
for row in rows:
    if row.get("tenantId") != expected_tenant or row.get("workspaceId") != expected_workspace:
        raise SystemExit(f"agent escaped expected scope {expected_tenant}/{expected_workspace}: {row}")
print(f"verified {expected_count} REST agent rows stay in {expected_tenant}/{expected_workspace}")
PY
}

assert_mcp_agents_scope() {
  local expected_count="$1"
  local expected_tenant="$2"
  local expected_workspace="$3"
  RESPONSE_BODY="$HTTP_BODY" python3 - "$expected_count" "$expected_tenant" "$expected_workspace" <<'PY'
import json
import os
import sys

expected_count = int(sys.argv[1])
expected_tenant = sys.argv[2]
expected_workspace = sys.argv[3]
doc = json.loads(os.environ["RESPONSE_BODY"])
if doc.get("error"):
    raise SystemExit(f"expected MCP result, got error: {doc['error']}")
rows = doc["result"]["structuredContent"]
if len(rows) != expected_count:
    raise SystemExit(f"expected {expected_count} MCP agent rows, got {len(rows)}: {rows}")
for row in rows:
    if row.get("tenantId") != expected_tenant or row.get("workspaceId") != expected_workspace:
        raise SystemExit(f"MCP agent escaped expected scope {expected_tenant}/{expected_workspace}: {row}")
print(f"verified {expected_count} MCP agent rows stay in {expected_tenant}/{expected_workspace}")
PY
}

assert_mcp_scope_error() {
  RESPONSE_BODY="$HTTP_BODY" python3 - <<'PY'
import json
import os

doc = json.loads(os.environ["RESPONSE_BODY"])
error = doc.get("error")
if not error:
    raise SystemExit(f"expected MCP error envelope, got: {doc}")
message = error.get("message", "")
if "outside authenticated admin scope" not in message:
    raise SystemExit(f"expected scope error message, got: {message!r}")
print("verified management MCP rejects out-of-scope arguments")
PY
}

need curl
need go
need python3

if port_in_use "$API_PORT"; then
  echo "API port $API_PORT is already in use" >&2
  exit 1
fi

mkdir -p "$LOG_DIR"
cd "$ROOT_DIR"

echo "AgentHarbor admin tenant boundary scenario"
echo "BASE_URL=$BASE_URL"
echo "RUN_ID=$RUN_ID"
echo "AGENT_HARBOR_ADMIN_IDENTITIES=platform=provided|role=platform_admin;east=provided|role=tenant_admin|tenant=${EAST_TENANT_ID}|workspace=${EAST_WORKSPACE_ID}"

AGENT_HARBOR_ADDR="$API_ADDR" \
AGENT_HARBOR_ADMIN_IDENTITIES="platform=${PLATFORM_KEY}|role=platform_admin;east=${EAST_KEY}|role=tenant_admin|tenant=${EAST_TENANT_ID}|workspace=${EAST_WORKSPACE_ID}" \
AGENT_HARBOR_SESSION_SECRET="admin-boundary-session-secret" \
AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=false \
AGENT_HARBOR_DATABASE_URL= \
AGENT_HARBOR_CREDENTIAL_KEY= \
  go run ./cmd/agent-harbor > "$LOG_DIR/api.log" 2>&1 &
PIDS+=("$!")

wait_http "API" "$BASE_URL/healthz"

request GET "/healthz" none
expect_status 200 "health check"

request POST "/api/v1/tenants" platform "$(json_body tenant "$ROOT_TENANT_ID" "" "Admin Boundary Root")"
expect_status 201 "create root tenant"
request POST "/api/v1/tenants" platform "$(json_body tenant "$EAST_TENANT_ID" "$ROOT_TENANT_ID" "Admin Boundary East")"
expect_status 201 "create east tenant"
request POST "/api/v1/tenants" platform "$(json_body tenant "$WEST_TENANT_ID" "$ROOT_TENANT_ID" "Admin Boundary West")"
expect_status 201 "create west tenant"

request POST "/api/v1/agents" platform "$(json_body agent "$EAST_TENANT_ID" "$EAST_WORKSPACE_ID" "East platform seeded agent")"
expect_status 201 "create platform east agent"
request POST "/api/v1/agents" platform "$(json_body agent "$WEST_TENANT_ID" "$WEST_WORKSPACE_ID" "West platform seeded agent")"
expect_status 201 "create platform west agent"

request GET "/api/v1/agents" east
expect_status 200 "east admin list scoped agents"
assert_agents_scope 1 "$EAST_TENANT_ID" "$EAST_WORKSPACE_ID"

request GET "/api/v1/agents?tenantId=${WEST_TENANT_ID}&workspaceId=${WEST_WORKSPACE_ID}" east
expect_status 403 "east admin blocked from west query"
echo "REST list query cannot widen outside scoped admin tenant"

request POST "/api/v1/agents" east "$(json_body agent "$EAST_TENANT_ID" "$EAST_WORKSPACE_ID" "East scoped admin created agent")"
expect_status 201 "east admin create in-scope agent"
echo "REST mutation accepts in-scope tenant admin write"

request POST "/api/v1/agents" east "$(json_body agent "$WEST_TENANT_ID" "$WEST_WORKSPACE_ID" "West scoped admin denied agent")"
expect_status 403 "east admin create west agent"
echo "REST mutation rejects out-of-scope tenant admin write"

request POST "/api/v1/management/mcp" east "$(json_body mcp-list-agents)"
expect_status 200 "east admin MCP scoped list"
assert_mcp_agents_scope 2 "$EAST_TENANT_ID" "$EAST_WORKSPACE_ID"

request POST "/api/v1/management/mcp" east "$(json_body mcp-list-agents "$WEST_TENANT_ID" "$WEST_WORKSPACE_ID")"
expect_status 200 "east admin MCP west query returns JSON-RPC error"
assert_mcp_scope_error

echo "admin tenant boundary scenario complete"
