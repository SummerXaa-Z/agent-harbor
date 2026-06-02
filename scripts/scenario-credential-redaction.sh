#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:9090}"
ADMIN_KEY="${ADMIN_KEY:-}"
WORKSPACE_ID="${WORKSPACE_ID:-scenario-workspace}"
RUN_ID="${RUN_ID:-credential-redaction-$(date +%Y%m%d%H%M%S)}"
SECRET_VALUE="${SECRET_VALUE:-Bearer credential-redaction-scenario-secret}"

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
    value = value[int(part)] if part.isdigit() else value[part]
print(value)
PY
}

json_body() {
  python3 - "$RUN_ID" "$WORKSPACE_ID" "$SECRET_VALUE" <<'PY'
import json
import sys

run_id, workspace_id, secret = sys.argv[1], sys.argv[2], sys.argv[3]
body = {
    "name": f"Credential Redaction Credentialed MCP {run_id}",
    "workspaceId": workspace_id,
    "channelType": "mcp",
    "status": "active",
    "channelConfig": {
        "endpoint": "https://api.example.com/mcp",
        "credentialHeaders": {"Authorization": "apiToken"},
    },
    "credentials": {"apiToken": secret},
}
print(json.dumps(body, separators=(",", ":")))
PY
}

assert_no_secret() {
  local label="$1"
  RESPONSE_BODY="$HTTP_BODY" SECRET_VALUE="$SECRET_VALUE" python3 - "$label" <<'PY'
import os
import sys

body = os.environ["RESPONSE_BODY"]
secret = os.environ["SECRET_VALUE"]
label = sys.argv[1]
if secret in body or '"credentials"' in body:
    raise SystemExit(f"{label} leaked credential material: {body}")
print(f"{label} redacted")
PY
}

need curl
need python3

echo "AgentHarbor credential redaction scenario"
echo "BASE_URL=$BASE_URL"
echo "RUN_ID=$RUN_ID"
if [[ -n "$ADMIN_KEY" ]]; then
  echo "ADMIN_KEY=provided"
else
  echo "ADMIN_KEY=not set"
fi

request GET "/healthz"
expect_status 200 "health check"

request POST "/api/v1/agents" "$(json_body)"
expect_status 201 "create credentialed agent"
assert_no_secret "create response"
AGENT_ID="$(json_get data.id)"
echo "created credentialed agent: $AGENT_ID"

request GET "/api/v1/agents/$AGENT_ID"
expect_status 200 "get credentialed agent"
assert_no_secret "get response"

request GET "/api/v1/agents?workspaceId=$WORKSPACE_ID"
expect_status 200 "list credentialed agents"
assert_no_secret "list response"

echo "credential redaction scenario complete"
