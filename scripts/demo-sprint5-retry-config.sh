#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:9090}"
ADMIN_KEY="${ADMIN_KEY:-}"
WORKSPACE_ID="${WORKSPACE_ID:-demo-workspace}"
RUN_ID="${RUN_ID:-sprint5-$(date +%Y%m%d%H%M%S)}"

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

json_body() {
  python3 - "$RUN_ID" "$WORKSPACE_ID" "$1" <<'PY'
import json
import sys

run_id, workspace_id, kind = sys.argv[1], sys.argv[2], sys.argv[3]
body = {
    "name": f"Sprint5 Retry {kind} {run_id}",
    "workspaceId": workspace_id,
    "channelType": "local",
    "status": "active",
}
if kind == "valid":
    body["channelConfig"] = {
        "retry": {
            "maxAttempts": 3,
            "backoffMs": 0,
            "statusCodes": [502, 503, 504],
        }
    }
elif kind == "bad-attempts":
    body["channelConfig"] = {"retry": {"maxAttempts": 5}}
elif kind == "bad-status":
    body["channelConfig"] = {"retry": {"statusCodes": [429]}}
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
retry = doc["data"]["channelConfig"]["retry"]
if retry.get("maxAttempts") != 3 or retry.get("backoffMs") != 0:
    raise SystemExit(f"unexpected retry config: {retry}")
print("retry config accepted")
PY
}

need curl
need python3

echo "AgentHarbor Sprint 5 retry config demo"
echo "BASE_URL=$BASE_URL"
echo "RUN_ID=$RUN_ID"
if [[ -n "$ADMIN_KEY" ]]; then
  echo "ADMIN_KEY=provided"
else
  echo "ADMIN_KEY=not set"
fi

request GET "/healthz"
expect_status 200 "health check"

request POST "/api/v1/agents" "$(json_body valid)"
expect_status 201 "create retry-configured agent"
assert_retry_shape

request POST "/api/v1/agents" "$(json_body bad-attempts)"
expect_status 400 "reject retry attempts above max"
echo "invalid retry attempts rejected"

request POST "/api/v1/agents" "$(json_body bad-status)"
expect_status 400 "reject non-5xx retry status"
echo "invalid retry status rejected"

echo "retry config demo complete"
