#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:9090}"
ADMIN_KEY="${ADMIN_KEY:-}"
RUN_ID="${RUN_ID:-runtime-metrics-$(date +%Y%m%d%H%M%S)}"
WORKSPACE_ID="${WORKSPACE_ID:-${RUN_ID}-workspace}"

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
    run_id, workspace_id, name, channel_type = sys.argv[2], sys.argv[3], sys.argv[4], sys.argv[5]
    body = {
        "name": f"Runtime Metrics {name} {run_id}",
        "workspaceId": workspace_id,
        "channelType": channel_type,
        "status": "active",
    }
elif kind == "agent-key":
    agent_id, run_id = sys.argv[2], sys.argv[3]
    body = {
        "agentId": agent_id,
        "name": f"runtime-metrics-key-{run_id}",
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
        "id": "metrics",
        "method": "tools/call",
        "params": {"name": "metrics.echo", "arguments": {"message": "hello"}},
    }
else:
    raise SystemExit(f"unknown body kind: {kind}")
print(json.dumps(body, separators=(",", ":")))
PY
}

assert_runtime_metrics() {
  RESPONSE_BODY="$HTTP_BODY" python3 <<'PY'
import json
import os

doc = json.loads(os.environ["RESPONSE_BODY"])
metrics = {item["id"]: item for item in doc.get("data", [])}
required = ["gateway_calls_total", "allowed_rate", "upstream_error_rate", "avg_latency_ms"]
missing = [key for key in required if key not in metrics]
if missing:
    raise SystemExit(f"missing runtime metrics: {missing}; got {sorted(metrics)}")
if metrics["gateway_calls_total"]["value"] < 2:
    raise SystemExit(f"expected at least two gateway calls, got {metrics['gateway_calls_total']}")
if metrics["allowed_rate"]["value"] != 50:
    raise SystemExit(f"expected isolated allowed_rate 50, got {metrics['allowed_rate']}")
if metrics["upstream_error_rate"]["value"] != 0:
    raise SystemExit(f"expected upstream_error_rate 0, got {metrics['upstream_error_rate']}")
print(
    "runtime metrics:",
    f"calls={metrics['gateway_calls_total']['value']}",
    f"allowed={metrics['allowed_rate']['value']}%",
    f"upstream_errors={metrics['upstream_error_rate']['value']}%",
)
PY
}

need curl
need python3

echo "AgentHarbor runtime metrics scenario"
echo "BASE_URL=$BASE_URL"
echo "RUN_ID=$RUN_ID"
echo "WORKSPACE_ID=$WORKSPACE_ID"
if [[ -n "$ADMIN_KEY" ]]; then
  echo "ADMIN_KEY=provided"
else
  echo "ADMIN_KEY=not set"
fi

request GET "/healthz"
expect_status 200 "health check"

request POST "/api/v1/agents" "$(json_body agent "$RUN_ID" "$WORKSPACE_ID" "Runtime Caller" "local")"
expect_status 201 "create local caller"
CALLER_ID="$(json_get data.id)"
echo "created caller: $CALLER_ID"

request POST "/api/v1/agents" "$(json_body agent "$RUN_ID" "$WORKSPACE_ID" "Runtime Target" "local")"
expect_status 201 "create governed target"
TARGET_ID="$(json_get data.id)"
echo "created target: $TARGET_ID"

request POST "/api/v1/agent-keys" "$(json_body agent-key "$CALLER_ID" "$RUN_ID")"
expect_status 201 "create agent key"
AGENT_KEY="$(json_get data.key)"
echo "created caller key"

request POST "/api/v1/mcp/agents/$TARGET_ID/rpc" "$(json_body mcp-call)" "$AGENT_KEY" "$RUN_ID"
expect_status 403 "denied call before grant"
echo "recorded denied call"

request POST "/api/v1/access-grants" "$(json_body grant "$CALLER_ID" "$TARGET_ID")"
expect_status 201 "create access grant"
echo "created grant"

request POST "/api/v1/mcp/agents/$TARGET_ID/rpc" "$(json_body mcp-call)" "$AGENT_KEY" "$RUN_ID"
expect_status 200 "allowed call after grant"
echo "recorded allowed call"

request GET "/api/v1/metrics/runtime?workspaceId=$WORKSPACE_ID"
expect_status 200 "runtime metrics"
assert_runtime_metrics

echo "runtime metrics scenario complete"
