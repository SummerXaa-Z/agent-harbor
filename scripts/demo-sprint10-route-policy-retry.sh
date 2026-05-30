#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:9090}"
ADMIN_KEY="${ADMIN_KEY:-}"
WORKSPACE_ID="${WORKSPACE_ID:-demo-workspace}"
RUN_ID="${RUN_ID:-sprint10-$(date +%Y%m%d%H%M%S)}"

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
        "name": f"Sprint10 {role} {run_id}",
        "workspaceId": workspace_id,
        "channelType": "local",
        "status": "active",
    }
elif kind == "policy":
    caller_id, target_id = sys.argv[2], sys.argv[3]
    body = {
        "name": "Sprint10 retry override",
        "callerAgentId": caller_id,
        "targetAgentId": target_id,
        "routeType": "mcp",
        "routeKey": "tools/call",
        "effect": "allow",
        "priority": 100,
        "retry": {
            "maxAttempts": 3,
            "backoffMs": 25,
            "statusCodes": [502, 503, 504],
        },
    }
elif kind == "patch-clear":
    body = {"retry": None}
elif kind == "bad-attempts":
    caller_id, target_id = sys.argv[2], sys.argv[3]
    body = {
        "name": "Sprint10 bad retry",
        "callerAgentId": caller_id,
        "targetAgentId": target_id,
        "routeType": "mcp",
        "routeKey": "tools/call",
        "effect": "allow",
        "retry": {"maxAttempts": 5},
    }
elif kind == "bad-status":
    body = {"retry": {"statusCodes": [429]}}
else:
    raise SystemExit(f"unknown body kind: {kind}")
print(json.dumps(body, separators=(",", ":")))
PY
}

assert_retry_shape() {
  RESPONSE_BODY="$HTTP_BODY" python3 <<'PY'
import json
import os

doc = json.loads(os.environ["RESPONSE_BODY"])
retry = doc["data"].get("retry")
if retry != {"maxAttempts": 3, "backoffMs": 25, "statusCodes": [502, 503, 504]}:
    raise SystemExit(f"unexpected retry shape: {retry}")
print("route policy retry shape verified")
PY
}

assert_retry_cleared() {
  RESPONSE_BODY="$HTTP_BODY" python3 <<'PY'
import json
import os

doc = json.loads(os.environ["RESPONSE_BODY"])
if "retry" in doc["data"]:
    raise SystemExit(f"retry should be cleared: {doc['data'].get('retry')}")
print("route policy retry clear verified")
PY
}

need curl
need python3

echo "AgentHarbor Sprint 10 route policy retry demo"
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

request POST "/api/v1/route-policies" "$(json_body policy "$CALLER_ID" "$TARGET_ID")"
expect_status 201 "create retry route policy"
POLICY_ID="$(json_get data.id)"
assert_retry_shape

request POST "/api/v1/route-policies" "$(json_body bad-attempts "$CALLER_ID" "$TARGET_ID")"
expect_status 400 "reject bad retry attempts"
echo "invalid retry attempts rejected"

request PATCH "/api/v1/route-policies/$POLICY_ID" "$(json_body bad-status)"
expect_status 400 "reject bad retry status"
echo "invalid retry status rejected"

request PATCH "/api/v1/route-policies/$POLICY_ID" "$(json_body patch-clear)"
expect_status 200 "clear retry override"
assert_retry_cleared

echo "route policy retry demo complete"
