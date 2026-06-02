#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:9090}"
ADMIN_KEY="${ADMIN_KEY:-}"
WORKSPACE_ID="${WORKSPACE_ID:-scenario-workspace}"
RUN_ID="${RUN_ID:-management-audit-$(date +%Y%m%d%H%M%S)}"
OLD_SECRET="Bearer management-audit-old-$RUN_ID"
NEW_SECRET="Bearer management-audit-new-$RUN_ID"

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
    run_id, workspace_id, secret = sys.argv[2], sys.argv[3], sys.argv[4]
    body = {
        "name": f"Management Audit Audited MCP {run_id}",
        "workspaceId": workspace_id,
        "channelType": "mcp",
        "status": "active",
        "channelConfig": {
            "endpoint": "https://api.example.com/mcp",
            "credentialHeaders": {"Authorization": "apiToken"},
        },
        "credentials": {"apiToken": secret},
    }
elif kind == "patch":
    run_id = sys.argv[2]
    body = {
        "name": f"Management Audit Audited MCP Updated {run_id}",
        "description": "patched by management audit scenario",
        "status": "draft",
    }
elif kind == "rotate":
    secret = sys.argv[2]
    body = {"credentials": {"apiToken": secret}}
else:
    raise SystemExit(f"unknown body kind: {kind}")
print(json.dumps(body, separators=(",", ":")))
PY
}

assert_no_secret_values() {
  local label="$1"
  if [[ "$HTTP_BODY" == *"$OLD_SECRET"* || "$HTTP_BODY" == *"$NEW_SECRET"* ]]; then
    echo "$label leaked credential values" >&2
    echo "$HTTP_BODY" >&2
    exit 1
  fi
}

assert_version() {
  local expected="$1"
  local label="$2"
  local got
  got="$(json_get data.credentialVersion)"
  if [[ "$got" != "$expected" ]]; then
    echo "expected $label credentialVersion $expected, got $got" >&2
    echo "$HTTP_BODY" >&2
    exit 1
  fi
}

assert_audit_events() {
  RESPONSE_BODY="$HTTP_BODY" python3 <<'PY'
import json
import os

doc = json.loads(os.environ["RESPONSE_BODY"])
events = doc["data"]
actions = [event["action"] for event in events]
expected = ["agent.created", "agent.updated", "agent.credentials_rotated"]
if actions != expected:
    raise SystemExit(f"unexpected audit actions: {actions}")
rotation = events[-1]
metadata = rotation.get("metadata") or {}
if metadata.get("credentialVersion") != 2:
    raise SystemExit(f"rotation metadata missing credentialVersion=2: {metadata}")
if metadata.get("credentialKeys") != ["apiToken"]:
    raise SystemExit(f"rotation metadata should expose only credential key names: {metadata}")
print("audit events verified")
PY
}

need curl
need python3

echo "AgentHarbor management audit scenario"
echo "BASE_URL=$BASE_URL"
echo "RUN_ID=$RUN_ID"
if [[ -n "$ADMIN_KEY" ]]; then
  echo "ADMIN_KEY=provided"
else
  echo "ADMIN_KEY=not set"
fi

request GET "/healthz"
expect_status 200 "health check"

request POST "/api/v1/agents" "$(json_body agent "$RUN_ID" "$WORKSPACE_ID" "$OLD_SECRET")"
expect_status 201 "create audited agent"
assert_no_secret_values "create response"
assert_version 1 "created agent"
AGENT_ID="$(json_get data.id)"
echo "created audited agent: $AGENT_ID"

request PATCH "/api/v1/agents/$AGENT_ID" "$(json_body patch "$RUN_ID")"
expect_status 200 "patch audited agent"
assert_no_secret_values "patch response"
assert_version 1 "patched agent"

request POST "/api/v1/agents/$AGENT_ID/credentials:rotate" "$(json_body rotate "$NEW_SECRET")"
expect_status 200 "rotate audited credentials"
assert_no_secret_values "rotate response"
assert_version 2 "rotated agent"

request GET "/api/v1/audit/events?resourceId=$AGENT_ID"
expect_status 200 "list resource audit events"
assert_no_secret_values "audit event list"
assert_audit_events

request GET "/api/v1/audit/events?action=agent.credentials_rotated&resourceType=agent&resourceId=$AGENT_ID"
expect_status 200 "list rotation audit event"
assert_no_secret_values "rotation audit event list"
ROTATION_COUNT="$(json_get data | python3 -c 'import ast,sys; print(len(ast.literal_eval(sys.stdin.read())))')"
if [[ "$ROTATION_COUNT" != "1" ]]; then
  echo "expected one filtered rotation audit event, got $ROTATION_COUNT" >&2
  echo "$HTTP_BODY" >&2
  exit 1
fi

echo "management audit scenario complete"
