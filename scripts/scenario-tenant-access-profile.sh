#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:9090}"
ADMIN_KEY="${ADMIN_KEY:-}"
RUN_ID="${RUN_ID:-tenant-access-profile-$(date +%Y%m%d%H%M%S)}"
ROOT_TENANT_ID="${ROOT_TENANT_ID:-tenant-root-${RUN_ID}}"
CHILD_TENANT_ID="${CHILD_TENANT_ID:-tenant-child-${RUN_ID}}"
GRANDCHILD_TENANT_ID="${GRANDCHILD_TENANT_ID:-tenant-grandchild-${RUN_ID}}"
WORKSPACE_ID="${WORKSPACE_ID:-ws-tenant-access-profile}"
MCP_ENDPOINT="${MCP_ENDPOINT:-}"
ALLOWED_TOOL="${ALLOWED_TOOL:-search_customer}"
SUBJECT_ID="${SUBJECT_ID:-user:tenant-access-profile}"

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
if kind == "tenant":
    tenant_id, parent_id, name = sys.argv[2], sys.argv[3], sys.argv[4]
    body = {"id": tenant_id, "name": name, "status": "active"}
    if parent_id:
        body["parentTenantId"] = parent_id
elif kind == "agent":
    tenant_id, workspace_id, name, channel_type = sys.argv[2], sys.argv[3], sys.argv[4], sys.argv[5]
    body = {
        "tenantId": tenant_id,
        "workspaceId": workspace_id,
        "name": name,
        "channelType": channel_type,
        "status": "active",
    }
    if len(sys.argv) > 6 and sys.argv[6]:
        body["channelConfig"] = {"endpoint": sys.argv[6]}
elif kind == "approve-scoped":
    tenant_id = sys.argv[2]
    body = {
        "discoveryStatus": "approved",
        "dataScopes": [{
            "dataDomain": "crm",
            "region": "us-east",
            "tenantFilter": f"tenant_id = '{tenant_id}'",
        }],
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
        "dataScopes": [{"field": "email"}],
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

assert_empty_profile() {
  RESPONSE_BODY="$HTTP_BODY" python3 - "$CHILD_TENANT_ID" "$GRANDCHILD_TENANT_ID" <<'PY'
import json
import os
import sys

child_id, grandchild_id = sys.argv[1], sys.argv[2]
profile = json.loads(os.environ["RESPONSE_BODY"])["data"]
if profile["tenant"]["id"] != child_id:
    raise SystemExit(f"tenant id mismatch: {profile['tenant']}")
scope_ids = [row["id"] for row in profile["scopeTenants"]]
if scope_ids != [child_id, grandchild_id]:
    raise SystemExit(f"scope tenants = {scope_ids}")
if profile["summary"]["grantCount"] != 0:
    raise SystemExit(f"expected no grants without MCP discovery, got {profile['summary']}")
print("empty tenant access profile verified")
PY
}

assert_profile_chain() {
  RESPONSE_BODY="$HTTP_BODY" python3 - "$CHILD_TENANT_ID" "$GRANDCHILD_TENANT_ID" "$TARGET_ID" "$ALLOWED_CAPABILITY_ID" <<'PY'
import json
import os
import sys

child_id, grandchild_id, target_id, capability_id = sys.argv[1:5]
profile = json.loads(os.environ["RESPONSE_BODY"])["data"]
if profile["tenant"]["id"] != child_id:
    raise SystemExit(f"tenant id mismatch: {profile['tenant']}")
scope_ids = [row["id"] for row in profile["scopeTenants"]]
if scope_ids != [child_id, grandchild_id]:
    raise SystemExit(f"scope tenants = {scope_ids}")
summary = profile["summary"]
expected_counts = {
    "grantCount": 1,
    "targetCount": 1,
    "capabilityCount": 1,
    "workspaceAssignmentCount": 1,
    "instanceAssignmentCount": 1,
}
for key, expected in expected_counts.items():
    if summary.get(key) != expected:
        raise SystemExit(f"summary[{key}]={summary.get(key)} want {expected}; summary={summary}")
if summary.get("recentAllowedTraceCount", 0) < 1:
    raise SystemExit(f"expected at least one allowed trace, summary={summary}")
grant = profile["grants"][0]
if grant.get("scopeStatus") != "valid":
    raise SystemExit(f"grant not valid: {grant}")
if grant["target"]["id"] != target_id or grant["capability"]["id"] != capability_id:
    raise SystemExit(f"target/capability mismatch: {grant}")
workspace = grant["workspaceAssignments"][0]
instance = workspace["instanceAssignments"][0]
scope = instance["effectiveInstanceDataScopes"][0]
expected_scope = {
    "dataDomain": "crm",
    "region": "us-east",
    "table": "accounts",
    "field": "email",
    "tenantFilter": f"tenant_id = '{child_id}'",
}
for key, expected in expected_scope.items():
    if scope.get(key) != expected:
        raise SystemExit(f"scope[{key}]={scope.get(key)!r} want {expected!r}; scope={scope}")
print("tenant access profile grant chain verified")
PY
}

need curl
need python3

echo "AgentHarbor tenant access profile scenario"
echo "BASE_URL=$BASE_URL"
echo "RUN_ID=$RUN_ID"

request GET "/healthz"
expect_status 200 "health check"

request POST "/api/v1/tenants" "$(json_body tenant "$ROOT_TENANT_ID" "" "Tenant Access Profile Root")"
expect_status 201 "create root tenant"
request POST "/api/v1/tenants" "$(json_body tenant "$CHILD_TENANT_ID" "$ROOT_TENANT_ID" "Tenant Access Profile Child")"
expect_status 201 "create child tenant"
request POST "/api/v1/tenants" "$(json_body tenant "$GRANDCHILD_TENANT_ID" "$CHILD_TENANT_ID" "Tenant Access Profile Grandchild")"
expect_status 201 "create grandchild tenant"

request POST "/api/v1/agents" "$(json_body agent "$CHILD_TENANT_ID" "$WORKSPACE_ID" "Tenant Access Profile Child Caller" "local")"
expect_status 201 "create child caller"
CALLER_ID="$(json_get data.id)"

request POST "/api/v1/agent-keys" "{\"agentId\":\"$CALLER_ID\",\"name\":\"tenant-access-profile key\"}"
expect_status 201 "create key"
AGENT_KEY="$(json_get data.key)"

if [[ -z "$MCP_ENDPOINT" ]]; then
  request POST "/api/v1/agents" "$(json_body agent "$ROOT_TENANT_ID" "$WORKSPACE_ID" "Tenant Access Profile Local Target" "local")"
  expect_status 201 "create local target"
  request GET "/api/v1/tenants/$CHILD_TENANT_ID/access-profile?traceLimit=0"
  expect_status 200 "fetch empty tenant access profile"
  assert_empty_profile
  echo "MCP_ENDPOINT is not set; empty profile path verified. Set MCP_ENDPOINT to verify the full grant chain."
  exit 0
fi

request POST "/api/v1/agents" "$(json_body agent "$ROOT_TENANT_ID" "$WORKSPACE_ID" "Tenant Access Profile Root MCP" "mcp" "$MCP_ENDPOINT")"
expect_status 201 "create root mcp target"
TARGET_ID="$(json_get data.id)"

request POST "/api/v1/targets/$TARGET_ID/capabilities:refresh"
expect_status 200 "refresh capabilities"
ALLOWED_CAPABILITY_ID="$(capability_id_for_key "$ALLOWED_TOOL")"

request PATCH "/api/v1/capabilities/$ALLOWED_CAPABILITY_ID" "$(json_body approve-scoped "$CHILD_TENANT_ID")"
expect_status 200 "approve scoped capability"

request POST "/api/v1/tenant-entitlements" "$(json_body entitlement "$CHILD_TENANT_ID" "$TARGET_ID" "$ALLOWED_CAPABILITY_ID")"
expect_status 201 "create child tenant entitlement"
ENTITLEMENT_ID="$(json_get data.id)"

request POST "/api/v1/workspace-assignments" "$(json_body workspace-assignment "$ENTITLEMENT_ID" "$WORKSPACE_ID")"
expect_status 201 "create workspace assignment"
WORKSPACE_ASSIGNMENT_ID="$(json_get data.id)"

request POST "/api/v1/instance-assignments" "$(json_body instance-assignment "$WORKSPACE_ASSIGNMENT_ID" "$CALLER_ID")"
expect_status 201 "create instance assignment"

request POST "/api/v1/mcp/agents/$TARGET_ID/rpc" "$(json_body tools-call "$ALLOWED_TOOL")" "$AGENT_KEY" "$RUN_ID-allowed" "$SUBJECT_ID"
expect_2xx "allow scoped tools/call"

request GET "/api/v1/tenants/$CHILD_TENANT_ID/access-profile?traceLimit=5"
expect_status 200 "fetch tenant access profile"
assert_profile_chain

echo "tenant access profile scenario complete"
