#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:9090}"
ADMIN_KEY="${ADMIN_KEY:-}"
WORKSPACE_ID="${WORKSPACE_ID:-demo-workspace}"
RUN_ID="${RUN_ID:-sprint9-$(date +%Y%m%d%H%M%S)}"

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
        "name": f"Sprint9 {role} {run_id}",
        "workspaceId": workspace_id,
        "channelType": "local",
        "status": "active",
    }
elif kind == "agent-key":
    agent_id, run_id = sys.argv[2], sys.argv[3]
    body = {
        "agentId": agent_id,
        "name": f"sprint9-key-{run_id}",
        "expiresInSeconds": 900,
    }
elif kind == "policy":
    name, caller_id, target_id, effect, priority = sys.argv[2:7]
    body = {
        "name": name,
        "callerAgentId": caller_id,
        "targetAgentId": target_id,
        "routeType": "mcp",
        "routeKey": "tools/call",
        "effect": effect,
        "priority": int(priority),
    }
elif kind == "patch-priority":
    body = {"priority": int(sys.argv[2])}
elif kind == "mcp-call":
    body = {"jsonrpc": "2.0", "id": "tools/call", "method": "tools/call"}
else:
    raise SystemExit(f"unknown body kind: {kind}")
print(json.dumps(body, separators=(",", ":")))
PY
}

assert_trace_reasons() {
  RESPONSE_BODY="$HTTP_BODY" python3 <<'PY'
import json
import os

doc = json.loads(os.environ["RESPONSE_BODY"])
reasons = [trace.get("reason", "") for trace in doc["data"]]
if "route policy allowed" not in reasons:
    raise SystemExit(f"missing allowed policy trace reason: {reasons}")
if "route policy denied" not in reasons:
    raise SystemExit(f"missing denied policy trace reason: {reasons}")
print("route policy trace reasons verified")
PY
}

need curl
need python3

echo "AgentHarbor Sprint 9 route policy demo"
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

request POST "/api/v1/agents" "$(json_body agent Target "$RUN_ID" "$WORKSPACE_ID")"
expect_status 201 "create target"
TARGET_ID="$(json_get data.id)"

request POST "/api/v1/agent-keys" "$(json_body agent-key "$CALLER_ID" "$RUN_ID")"
expect_status 201 "create caller key"
AGENT_KEY="$(json_get data.key)"

request POST "/api/v1/mcp/agents/$TARGET_ID/rpc" "$(json_body mcp-call)" "$AGENT_KEY" "$RUN_ID"
expect_status 403 "call before policy"

request POST "/api/v1/route-policies" "$(json_body policy "Sprint9 allow calls $RUN_ID" "$CALLER_ID" "$TARGET_ID" allow 100)"
expect_status 201 "create allow route policy"
ALLOW_POLICY_ID="$(json_get data.id)"

request POST "/api/v1/mcp/agents/$TARGET_ID/rpc" "$(json_body mcp-call)" "$AGENT_KEY" "$RUN_ID"
expect_status 200 "call after allow policy"

request POST "/api/v1/route-policies" "$(json_body policy "Sprint9 deny calls $RUN_ID" "$CALLER_ID" "$TARGET_ID" deny 200)"
expect_status 201 "create higher priority deny policy"
DENY_POLICY_ID="$(json_get data.id)"

request POST "/api/v1/mcp/agents/$TARGET_ID/rpc" "$(json_body mcp-call)" "$AGENT_KEY" "$RUN_ID"
expect_status 403 "call after deny policy"

request PATCH "/api/v1/route-policies/$DENY_POLICY_ID" "$(json_body patch-priority 50)"
expect_status 200 "lower deny priority"

request POST "/api/v1/mcp/agents/$TARGET_ID/rpc" "$(json_body mcp-call)" "$AGENT_KEY" "$RUN_ID"
expect_status 200 "allow policy wins after deny priority lowered"

request DELETE "/api/v1/route-policies/$ALLOW_POLICY_ID"
expect_status 200 "disable allow policy"

request POST "/api/v1/mcp/agents/$TARGET_ID/rpc" "$(json_body mcp-call)" "$AGENT_KEY" "$RUN_ID"
expect_status 403 "remaining deny policy blocks call"

request GET "/api/v1/route-policies?workspaceId=$WORKSPACE_ID"
expect_status 200 "list route policies"
POLICY_COUNT="$(json_get data | python3 -c 'import ast,sys; print(len(ast.literal_eval(sys.stdin.read())))')"
if [[ "$POLICY_COUNT" -lt 2 ]]; then
  echo "expected at least two route policies, got $POLICY_COUNT" >&2
  echo "$HTTP_BODY" >&2
  exit 1
fi

request GET "/api/v1/audit/traces?runId=$RUN_ID"
expect_status 200 "list route policy traces"
assert_trace_reasons

echo "route policy demo complete"
