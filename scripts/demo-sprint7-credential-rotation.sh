#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:9090}"
ADMIN_KEY="${ADMIN_KEY:-}"
WORKSPACE_ID="${WORKSPACE_ID:-demo-workspace}"
RUN_ID="${RUN_ID:-sprint7-$(date +%Y%m%d%H%M%S)}"
OLD_SECRET="Bearer sprint7-old-$RUN_ID"
NEW_SECRET="Bearer sprint7-new-$RUN_ID"

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
        "name": f"Sprint7 Credentialed MCP {run_id}",
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
        "name": f"Sprint7 Patched MCP {run_id}",
        "description": "patched by sprint7 demo",
        "ownerId": "platform-team",
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

assert_redacted() {
  local label="$1"
  if [[ "$HTTP_BODY" == *"$OLD_SECRET"* || "$HTTP_BODY" == *"$NEW_SECRET"* || "$HTTP_BODY" == *"credentials"* ]]; then
    echo "$label leaked credential material" >&2
    echo "$HTTP_BODY" >&2
    exit 1
  fi
}

assert_patched_agent() {
  RESPONSE_BODY="$HTTP_BODY" RUN_ID="$RUN_ID" python3 <<'PY'
import json
import os

doc = json.loads(os.environ["RESPONSE_BODY"])
data = doc["data"]
if data["name"] != f"Sprint7 Patched MCP {os.environ['RUN_ID']}":
    raise SystemExit(f"unexpected patched name: {data}")
if data["description"] != "patched by sprint7 demo" or data["ownerId"] != "platform-team" or data["status"] != "draft":
    raise SystemExit(f"unexpected patched metadata: {data}")
print("patch response verified")
PY
}

need curl
need python3

echo "AgentHarbor Sprint 7 credential rotation demo"
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
expect_status 201 "create credentialed agent"
assert_redacted "create response"
AGENT_ID="$(json_get data.id)"
echo "created credentialed agent: $AGENT_ID"

request PATCH "/api/v1/agents/$AGENT_ID" "$(json_body patch "$RUN_ID")"
expect_status 200 "patch agent"
assert_redacted "patch response"
assert_patched_agent

request POST "/api/v1/agents/$AGENT_ID/credentials:rotate" "$(json_body rotate "$NEW_SECRET")"
expect_status 200 "rotate credentials"
assert_redacted "rotate response"
echo "credential rotation response redacted"

request GET "/api/v1/agents/$AGENT_ID"
expect_status 200 "get rotated agent"
assert_redacted "get response"
echo "get response remains redacted"

echo "credential rotation demo complete"
