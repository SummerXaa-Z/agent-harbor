#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:9090}"
ADMIN_KEY="${ADMIN_KEY:-}"
WORKSPACE_ID="${WORKSPACE_ID:-scenario-workspace}"
RUN_ID="${RUN_ID:-registry-cleanup-$(date +%Y%m%d%H%M%S)}"

HTTP_STATUS=""
HTTP_BODY=""

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing dependency: $1" >&2
    exit 1
  fi
}

request() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  local bearer="${4:-}"
  local run_id="${5:-}"
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

  if [[ -n "$ADMIN_KEY" ]]; then
    args+=(-H "X-Admin-Key: $ADMIN_KEY")
  fi
  if [[ -n "$bearer" ]]; then
    args+=(-H "Authorization: Bearer $bearer")
  fi
  if [[ -n "$run_id" ]]; then
    args+=(-H "X-Run-Id: $run_id")
  fi
  if [[ -n "$body" ]]; then
    args+=(-d "$body")
  fi

  if ! HTTP_STATUS="$(curl "${args[@]}")"; then
    rm -f "$tmp"
    echo "curl failed for $method $path" >&2
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
    exit 1
  fi
}

json_get() {
  local path="$1"
  RESPONSE_BODY="$HTTP_BODY" python3 - "$path" <<'PY'
import json
import os
import sys

value = json.loads(os.environ["RESPONSE_BODY"])
for part in sys.argv[1].split("."):
    if part.isdigit():
        value = value[int(part)]
    else:
        value = value[part]
print(value)
PY
}

json_body() {
  python3 - "$@" <<'PY'
import json
import sys

kind = sys.argv[1]
if kind == "agent":
    role, run_id, workspace_id = sys.argv[2], sys.argv[3], sys.argv[4]
    body = {
        "name": f"Registry Cleanup {role} {run_id}",
        "workspaceId": workspace_id,
        "channelType": "local",
        "status": "active",
    }
elif kind == "agent-key":
    agent_id, run_id = sys.argv[2], sys.argv[3]
    body = {
        "agentId": agent_id,
        "name": f"registry-cleanup-key-{run_id}",
        "expiresInSeconds": 900,
    }
elif kind == "grant":
    caller_id, target_id = sys.argv[2], sys.argv[3]
    body = {
        "callerAgentId": caller_id,
        "targetAgentId": target_id,
        "routeType": "mcp",
        "routeKey": "tools/call",
    }
elif kind == "mcp-call":
    body = {
        "jsonrpc": "2.0",
        "id": "registry-cleanup",
        "method": "tools/call",
        "params": {"name": "scenario.cleanup", "arguments": {"message": "hello"}},
    }
else:
    raise SystemExit(f"unknown body kind: {kind}")
print(json.dumps(body, separators=(",", ":")))
PY
}

assert_scoped_lists() {
  local caller_id="$1"
  local target_id="$2"
  RESPONSE_BODY="$HTTP_BODY" python3 - "$caller_id" "$target_id" <<'PY'
import json
import os
import sys

doc = json.loads(os.environ["RESPONSE_BODY"])
agents = doc.get("data", [])
by_id = {agent.get("id"): agent for agent in agents}
for expected in sys.argv[1:]:
    if expected not in by_id:
        raise SystemExit(f"missing scoped agent {expected}; got {list(by_id)}")
if by_id[sys.argv[1]].get("status") != "disabled":
    raise SystemExit(f"caller should be disabled; got {by_id[sys.argv[1]]}")
print(f"scoped agents include disabled caller and target: {len(agents)} rows")
PY
}

need curl
need python3

echo "AgentHarbor registry cleanup scenario"
echo "BASE_URL=$BASE_URL"
echo "RUN_ID=$RUN_ID"
if [[ -n "$ADMIN_KEY" ]]; then
  echo "ADMIN_KEY=provided"
else
  echo "ADMIN_KEY=not set"
fi

request GET "/healthz"
expect_status 200 "health check"

request POST "/api/v1/agents" "$(json_body agent Caller "$RUN_ID" "$WORKSPACE_ID")"
expect_status 201 "create caller"
CALLER_ID="$(json_get data.id)"
echo "created caller: $CALLER_ID"

request POST "/api/v1/agents" "$(json_body agent Target "$RUN_ID" "$WORKSPACE_ID")"
expect_status 201 "create target"
TARGET_ID="$(json_get data.id)"
echo "created target: $TARGET_ID"

request POST "/api/v1/agent-keys" "$(json_body agent-key "$CALLER_ID" "$RUN_ID")"
expect_status 201 "create agent key"
AGENT_KEY="$(json_get data.key)"
echo "created agent key: $(json_get data.prefix)..."

request POST "/api/v1/access-grants" "$(json_body grant "$CALLER_ID" "$TARGET_ID")"
expect_status 201 "create access grant"
GRANT_ID="$(json_get data.id)"
echo "created access grant: $GRANT_ID"

request POST "/api/v1/mcp/agents/$TARGET_ID/rpc" "$(json_body mcp-call)" "$AGENT_KEY" "$RUN_ID"
expect_status 200 "allowed call before revoke"
echo "allowed call before revoke"

request DELETE "/api/v1/access-grants/$GRANT_ID"
expect_status 200 "revoke access grant"
echo "revoked access grant: $GRANT_ID"

request POST "/api/v1/mcp/agents/$TARGET_ID/rpc" "$(json_body mcp-call)" "$AGENT_KEY" "$RUN_ID"
expect_status 403 "denied call after grant revoke"
echo "denied call after grant revoke"

request POST "/api/v1/access-grants" "$(json_body grant "$CALLER_ID" "$TARGET_ID")"
expect_status 201 "recreate access grant"
echo "recreated access grant: $(json_get data.id)"

request DELETE "/api/v1/agents/$CALLER_ID"
expect_status 200 "disable caller agent"
echo "disabled caller agent: $CALLER_ID"

request POST "/api/v1/mcp/agents/$TARGET_ID/rpc" "$(json_body mcp-call)" "$AGENT_KEY" "$RUN_ID"
expect_status 401 "caller key after agent disable"
echo "disabled caller key rejected"

request GET "/api/v1/agents?tenantId=default&workspaceId=$WORKSPACE_ID"
expect_status 200 "scoped agent list"
assert_scoped_lists "$CALLER_ID" "$TARGET_ID"

echo "cleanup scenario complete"
