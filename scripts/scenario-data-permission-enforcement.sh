#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:9090}"
ADMIN_KEY="${ADMIN_KEY:-}"
WORKSPACE_ID="${WORKSPACE_ID:-scenario-workspace}"
TENANT_ID="${TENANT_ID:-default}"
RUN_ID="${RUN_ID:-data-permission-$(date +%Y%m%d%H%M%S)}"
MCP_ENDPOINT="${MCP_ENDPOINT:-}"
ALLOWED_TOOL="${ALLOWED_TOOL:-search_customer}"
SUBJECT_ID="${SUBJECT_ID:-user:data-permission}"

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
  local subject_id="${6:-}"
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
  if [[ -n "$subject_id" ]]; then
    args+=(-H "X-AgentHarbor-Subject-Id: $subject_id")
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

expect_2xx() {
  local label="$1"
  if [[ "$HTTP_STATUS" != 2* ]]; then
    echo "expected $label 2xx status, got $HTTP_STATUS" >&2
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
if kind == "caller":
    tenant_id, workspace_id, run_id = sys.argv[2], sys.argv[3], sys.argv[4]
    body = {
        "tenantId": tenant_id,
        "workspaceId": workspace_id,
        "name": f"Data Permission Caller {run_id}",
        "channelType": "local",
        "status": "active",
    }
elif kind == "target":
    tenant_id, workspace_id, endpoint, run_id = sys.argv[2], sys.argv[3], sys.argv[4], sys.argv[5]
    body = {
        "tenantId": tenant_id,
        "workspaceId": workspace_id,
        "name": f"Data Permission MCP Target {run_id}",
        "channelType": "mcp",
        "status": "active",
        "channelConfig": {
            "endpoint": endpoint,
            "headers": {"X-AgentHarbor-Context": "configured-spoof"},
        },
    }
elif kind == "approve-scoped":
    body = {
        "discoveryStatus": "approved",
        "dataScopes": [{
            "dataDomain": "crm",
            "region": "us-east",
            "tenantFilter": f"tenant_id = '{sys.argv[2]}'",
        }],
    }
elif kind == "bad-entitlement":
    tenant_id, target_id, capability_id = sys.argv[2], sys.argv[3], sys.argv[4]
    body = {
        "tenantId": tenant_id,
        "targetId": target_id,
        "capabilityId": capability_id,
        "effect": "allow",
        "status": "enabled",
        "dataScopes": [{"dataDomain": "crm", "region": "eu-west"}],
    }
elif kind == "entitlement":
    tenant_id, target_id, capability_id = sys.argv[2], sys.argv[3], sys.argv[4]
    body = {
        "tenantId": tenant_id,
        "targetId": target_id,
        "capabilityId": capability_id,
        "effect": "allow",
        "status": "enabled",
    }
elif kind == "bad-workspace-assignment":
    entitlement_id, workspace_id = sys.argv[2], sys.argv[3]
    body = {
        "tenantEntitlementId": entitlement_id,
        "workspaceId": workspace_id,
        "effect": "allow",
        "status": "enabled",
        "dataScopes": [{"dataDomain": "crm", "region": "eu-west"}],
    }
elif kind == "workspace-assignment":
    entitlement_id, workspace_id = sys.argv[2], sys.argv[3]
    body = {
        "tenantEntitlementId": entitlement_id,
        "workspaceId": workspace_id,
        "effect": "allow",
        "status": "enabled",
        "dataScopes": [{"table": "accounts"}],
    }
elif kind == "instance-assignment":
    workspace_assignment_id, caller_id = sys.argv[2], sys.argv[3]
    body = {
        "workspaceAssignmentId": workspace_assignment_id,
        "callerInstanceId": caller_id,
        "subjectSelector": "user:*",
        "effect": "allow",
        "status": "enabled",
    }
elif kind == "tools-call":
    tool = sys.argv[2]
    body = {"jsonrpc": "2.0", "id": f"call-{tool}", "method": "tools/call", "params": {"name": tool, "arguments": {}}}
else:
    raise SystemExit(f"unknown body kind: {kind}")
print(json.dumps(body, separators=(",", ":")))
PY
}

capability_id_for_key() {
  local key="$1"
  RESPONSE_BODY="$HTTP_BODY" python3 - "$key" <<'PY'
import json
import os
import sys

doc = json.loads(os.environ["RESPONSE_BODY"])
key = sys.argv[1]
for capability in doc["data"]:
    if capability.get("key") == key:
        print(capability["id"])
        raise SystemExit(0)
raise SystemExit(f"capability {key!r} not discovered")
PY
}

assert_trace_data_scopes() {
  RESPONSE_BODY="$HTTP_BODY" python3 - "$TENANT_ID" <<'PY'
import json
import os
import sys

tenant_id = sys.argv[1]
doc = json.loads(os.environ["RESPONSE_BODY"])
traces = doc["data"]
if not traces:
    raise SystemExit("expected at least one trace")
trace = traces[0]
scopes = trace.get("dataScopes") or []
if len(scopes) != 1:
    raise SystemExit(f"expected one effective data scope, got {scopes}")
scope = scopes[0]
expected = {
    "dataDomain": "crm",
    "region": "us-east",
    "table": "accounts",
    "tenantFilter": f"tenant_id = '{tenant_id}'",
}
for key, value in expected.items():
    if scope.get(key) != value:
        raise SystemExit(f"scope[{key!r}] = {scope.get(key)!r}, want {value!r}; scope={scope}")
print("effective data scopes verified")
PY
}

need curl
need python3

echo "AgentHarbor data permission enforcement scenario"
echo "BASE_URL=$BASE_URL"
echo "RUN_ID=$RUN_ID"
if [[ -n "$ADMIN_KEY" ]]; then
  echo "ADMIN_KEY=provided"
else
  echo "ADMIN_KEY=not set"
fi

if [[ -z "$MCP_ENDPOINT" ]]; then
  echo "MCP_ENDPOINT is not set; skipping data permission scenario because capability discovery needs a safe test MCP HTTP endpoint."
  echo "Set MCP_ENDPOINT to an endpoint exposing $ALLOWED_TOOL to run the full scenario."
  exit 0
fi

request GET "/healthz"
expect_status 200 "health check"

request POST "/api/v1/agents" "$(json_body caller "$TENANT_ID" "$WORKSPACE_ID" "$RUN_ID")"
expect_status 201 "create caller"
CALLER_ID="$(json_get data.id)"

request POST "/api/v1/agent-keys" "{\"agentId\":\"$CALLER_ID\",\"name\":\"data-permission key\"}"
expect_status 201 "create key"
AGENT_KEY="$(json_get data.key)"

request POST "/api/v1/agents" "$(json_body target "$TENANT_ID" "$WORKSPACE_ID" "$MCP_ENDPOINT" "$RUN_ID")"
expect_status 201 "create mcp target"
TARGET_ID="$(json_get data.id)"

request POST "/api/v1/targets/$TARGET_ID/capabilities:refresh"
expect_status 200 "refresh capabilities"
ALLOWED_CAPABILITY_ID="$(capability_id_for_key "$ALLOWED_TOOL")"
echo "discovered capability: $ALLOWED_CAPABILITY_ID"

request PATCH "/api/v1/capabilities/$ALLOWED_CAPABILITY_ID" "$(json_body approve-scoped "$TENANT_ID")"
expect_status 200 "approve scoped capability"

request POST "/api/v1/tenant-entitlements" "$(json_body bad-entitlement "$TENANT_ID" "$TARGET_ID" "$ALLOWED_CAPABILITY_ID")"
expect_status 400 "reject tenant entitlement scope expansion"
echo "tenant entitlement scope expansion rejected"

request POST "/api/v1/tenant-entitlements" "$(json_body entitlement "$TENANT_ID" "$TARGET_ID" "$ALLOWED_CAPABILITY_ID")"
expect_status 201 "create tenant entitlement"
ENTITLEMENT_ID="$(json_get data.id)"

request POST "/api/v1/workspace-assignments" "$(json_body bad-workspace-assignment "$ENTITLEMENT_ID" "$WORKSPACE_ID")"
expect_status 400 "reject workspace assignment scope expansion"
echo "workspace assignment scope expansion rejected"

request POST "/api/v1/workspace-assignments" "$(json_body workspace-assignment "$ENTITLEMENT_ID" "$WORKSPACE_ID")"
expect_status 201 "create workspace assignment"
WORKSPACE_ASSIGNMENT_ID="$(json_get data.id)"

request POST "/api/v1/instance-assignments" "$(json_body instance-assignment "$WORKSPACE_ASSIGNMENT_ID" "$CALLER_ID")"
expect_status 201 "create instance assignment"

request POST "/api/v1/mcp/agents/$TARGET_ID/rpc" "$(json_body tools-call "$ALLOWED_TOOL")" "$AGENT_KEY" "$RUN_ID-allowed" "$SUBJECT_ID"
expect_2xx "allow scoped tools/call"
echo "scoped tool call allowed"

request GET "/api/v1/audit/traces?runId=$RUN_ID-allowed"
expect_status 200 "list traces"
assert_trace_data_scopes

echo "data permission enforcement scenario complete"
