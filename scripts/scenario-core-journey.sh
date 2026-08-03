#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:9090}"
ADMIN_KEY="${ADMIN_KEY:-}"
RUN_ID="${RUN_ID:-core-journey-$(date +%Y%m%d%H%M%S)}"
ROOT_TENANT_ID="${ROOT_TENANT_ID:-tenant-root-${RUN_ID}}"
CHILD_TENANT_ID="${CHILD_TENANT_ID:-tenant-child-${RUN_ID}}"
GRANDCHILD_TENANT_ID="${GRANDCHILD_TENANT_ID:-tenant-grandchild-${RUN_ID}}"
WORKSPACE_ID="${WORKSPACE_ID:-ws-core-journey}"
MOCK_MCP_HOST="${MOCK_MCP_HOST:-127.0.0.1}"
MOCK_MCP_PORT="${MOCK_MCP_PORT:-8787}"
MCP_ENDPOINT="${MCP_ENDPOINT:-http://${MOCK_MCP_HOST}:${MOCK_MCP_PORT}/mcp}"
START_MOCK_MCP="${START_MOCK_MCP:-true}"
ALLOWED_TOOL="${ALLOWED_TOOL:-search_customer}"
DENIED_TOOL="${DENIED_TOOL:-export_contracts}"
SUBJECT_ID="${SUBJECT_ID:-user:core-journey}"

HTTP_STATUS=""
HTTP_BODY=""
MOCK_MCP_PID=""

cleanup() {
  if [[ -n "$MOCK_MCP_PID" ]]; then
    kill "$MOCK_MCP_PID" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

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
    if [[ "$HTTP_BODY" == *"endpoint host is not allowed"* ]]; then
      echo >&2
      echo "local MCP endpoints require: AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true make run" >&2
    fi
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
elif kind == "narrow-capability-scope":
    tenant_id = sys.argv[2]
    body = {
        "dataScopes": [{
            "dataDomain": "crm",
            "region": "us-east",
            "table": "contacts",
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
elif kind == "tools-list":
    body = {"jsonrpc": "2.0", "id": "tools-list", "method": "tools/list"}
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

assert_tools_list_filtered() {
  RESPONSE_BODY="$HTTP_BODY" python3 - "$ALLOWED_TOOL" "$DENIED_TOOL" <<'PY'
import json
import os
import sys

doc = json.loads(os.environ["RESPONSE_BODY"])
allowed, denied = sys.argv[1], sys.argv[2]
tools = {tool["name"] for tool in doc["result"]["tools"]}
if allowed not in tools:
    raise SystemExit(f"allowed tool {allowed!r} missing from tools/list: {tools}")
if denied in tools:
    raise SystemExit(f"denied tool {denied!r} should have been filtered: {tools}")
print("tools/list filtering verified")
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
if summary.get("recentAllowedTraceCount", 0) < 1 or summary.get("recentDeniedTraceCount", 0) < 1:
    raise SystemExit(f"expected allowed and denied runtime records, summary={summary}")
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

assert_data_scope_denied_trace() {
  RESPONSE_BODY="$HTTP_BODY" python3 - "$ALLOWED_CAPABILITY_ID" <<'PY'
import json
import os
import sys

capability_id = sys.argv[1]
traces = json.loads(os.environ["RESPONSE_BODY"])["data"]
if len(traces) != 1:
    raise SystemExit(f"expected one data-scope denial trace, got {traces}")
trace = traces[0]
expected_reason = "workspace assignment data scopes exceed tenant entitlement boundary"
if trace.get("decision") != "denied" or trace.get("reason") != expected_reason:
    raise SystemExit(f"unexpected data-scope denial trace: {trace}")
if trace.get("capabilityId") != capability_id:
    raise SystemExit(f"denial capability mismatch: {trace}")
if not trace.get("entitlementId") or not trace.get("workspaceAssignmentId"):
    raise SystemExit(f"denial is missing matched grant evidence: {trace}")
print(f"data-scope denial reason: {trace['reason']}")
PY
}

assert_data_scope_mismatch_profile() {
  RESPONSE_BODY="$HTTP_BODY" python3 - "$ALLOWED_CAPABILITY_ID" <<'PY'
import json
import os
import sys

capability_id = sys.argv[1]
profile = json.loads(os.environ["RESPONSE_BODY"])["data"]
grants = [grant for grant in profile["grants"] if grant.get("capability", {}).get("id") == capability_id]
if len(grants) != 1:
    raise SystemExit(f"expected one matching grant in access profile, got {grants}")
grant = grants[0]
tenant_scopes = grant.get("effectiveTenantDataScopes", [])
if len(tenant_scopes) != 1 or tenant_scopes[0].get("table") != "contacts":
    raise SystemExit(f"expected narrowed effective tenant scope, got {tenant_scopes}")
workspace = grant["workspaceAssignments"][0]
expected_reason = "workspace assignment dataScopes exceed tenant entitlement dataScopes"
if workspace.get("scopeStatus") != "invalid" or workspace.get("scopeReason") != expected_reason:
    raise SystemExit(f"workspace scope should be invalid after narrowing: {workspace}")
requested_scopes = workspace["workspaceAssignment"].get("dataScopes", [])
if len(requested_scopes) != 1 or requested_scopes[0].get("table") != "accounts":
    raise SystemExit(f"expected original workspace scope evidence, got {requested_scopes}")
evidence = {
    "effectiveTenantDataScopes": tenant_scopes,
    "workspaceRequestedDataScopes": requested_scopes,
    "workspaceScopeStatus": workspace["scopeStatus"],
    "workspaceScopeReason": workspace["scopeReason"],
}
print("data-scope evidence: " + json.dumps(evidence, ensure_ascii=False, separators=(",", ":")))
PY
}

start_mock_mcp() {
  if [[ "$START_MOCK_MCP" != "true" ]]; then
    return
  fi
  if curl -fsS "http://${MOCK_MCP_HOST}:${MOCK_MCP_PORT}/healthz" >/dev/null 2>&1; then
    echo "mock MCP server already running on ${MOCK_MCP_HOST}:${MOCK_MCP_PORT}"
    return
  fi
  scripts/mock-mcp-server.py --host "$MOCK_MCP_HOST" --port "$MOCK_MCP_PORT" &
  MOCK_MCP_PID="$!"
  for _ in $(seq 1 30); do
    if curl -fsS "http://${MOCK_MCP_HOST}:${MOCK_MCP_PORT}/healthz" >/dev/null 2>&1; then
      echo "started mock MCP server on ${MCP_ENDPOINT}"
      return
    fi
    sleep 0.2
  done
  echo "mock MCP server did not become ready" >&2
  exit 1
}

need curl
need python3

echo "AgentHarbor core journey scenario"
echo "BASE_URL=$BASE_URL"
echo "MCP_ENDPOINT=$MCP_ENDPOINT"
echo "RUN_ID=$RUN_ID"
if [[ -n "$ADMIN_KEY" ]]; then
  echo "ADMIN_KEY=provided"
else
  echo "ADMIN_KEY=not set"
fi

start_mock_mcp

request GET "/healthz"
expect_status 200 "AgentHarbor health check"

request POST "/api/v1/tenants" "$(json_body tenant "$ROOT_TENANT_ID" "" "Core Journey Root")"
expect_status 201 "create root tenant"
request POST "/api/v1/tenants" "$(json_body tenant "$CHILD_TENANT_ID" "$ROOT_TENANT_ID" "Core Journey Team")"
expect_status 201 "create child tenant"
request POST "/api/v1/tenants" "$(json_body tenant "$GRANDCHILD_TENANT_ID" "$CHILD_TENANT_ID" "Core Journey Project")"
expect_status 201 "create grandchild tenant"
echo "created tenant tree: $ROOT_TENANT_ID -> $CHILD_TENANT_ID -> $GRANDCHILD_TENANT_ID"

request POST "/api/v1/agents" "$(json_body agent "$CHILD_TENANT_ID" "$WORKSPACE_ID" "Core Journey Caller" "local")"
expect_status 201 "create caller agent"
CALLER_ID="$(json_get data.id)"

request POST "/api/v1/agent-keys" "{\"agentId\":\"$CALLER_ID\",\"name\":\"core journey key\"}"
expect_status 201 "create caller key"
AGENT_KEY="$(json_get data.key)"

request POST "/api/v1/agents" "$(json_body agent "$ROOT_TENANT_ID" "$WORKSPACE_ID" "Core Journey MCP Target" "mcp" "$MCP_ENDPOINT")"
expect_status 201 "create MCP target"
TARGET_ID="$(json_get data.id)"

request POST "/api/v1/targets/$TARGET_ID/capabilities:refresh"
expect_status 200 "discover MCP capabilities"
ALLOWED_CAPABILITY_ID="$(capability_id_for_key "$ALLOWED_TOOL")"
DENIED_CAPABILITY_ID="$(capability_id_for_key "$DENIED_TOOL")"
echo "discovered capabilities: allow=$ALLOWED_CAPABILITY_ID deny=$DENIED_CAPABILITY_ID"

request PATCH "/api/v1/capabilities/$ALLOWED_CAPABILITY_ID" "$(json_body approve-scoped "$CHILD_TENANT_ID")"
expect_status 200 "approve allowed capability"

request POST "/api/v1/tenant-entitlements" "$(json_body entitlement "$CHILD_TENANT_ID" "$TARGET_ID" "$ALLOWED_CAPABILITY_ID")"
expect_status 201 "create tenant entitlement"
ENTITLEMENT_ID="$(json_get data.id)"

request POST "/api/v1/workspace-assignments" "$(json_body workspace-assignment "$ENTITLEMENT_ID" "$WORKSPACE_ID")"
expect_status 201 "create workspace assignment"
WORKSPACE_ASSIGNMENT_ID="$(json_get data.id)"

request POST "/api/v1/instance-assignments" "$(json_body instance-assignment "$WORKSPACE_ASSIGNMENT_ID" "$CALLER_ID")"
expect_status 201 "create caller instance assignment"

request POST "/api/v1/mcp/agents/$TARGET_ID/rpc" "$(json_body tools-list)" "$AGENT_KEY" "$RUN_ID-tools-list" "$SUBJECT_ID"
expect_status 200 "filtered tools/list"
assert_tools_list_filtered

request POST "/api/v1/mcp/agents/$TARGET_ID/rpc" "$(json_body tools-call "$DENIED_TOOL")" "$AGENT_KEY" "$RUN_ID-denied" "$SUBJECT_ID"
expect_status 403 "deny unassigned tools/call"
echo "denied unassigned tool: $DENIED_TOOL"

request POST "/api/v1/mcp/agents/$TARGET_ID/rpc" "$(json_body tools-call "$ALLOWED_TOOL")" "$AGENT_KEY" "$RUN_ID-allowed" "$SUBJECT_ID"
expect_2xx "allow assigned tools/call"
echo "allowed assigned tool: $ALLOWED_TOOL"

request GET "/api/v1/tenants/$CHILD_TENANT_ID/access-profile?traceLimit=10"
expect_status 200 "fetch tenant access profile"
assert_profile_chain

request PATCH "/api/v1/capabilities/$ALLOWED_CAPABILITY_ID" "$(json_body narrow-capability-scope "$CHILD_TENANT_ID")"
expect_status 200 "narrow capability data scope"

DATA_SCOPE_MISMATCH_RUN_ID="$RUN_ID-data-scope-mismatch"
request POST "/api/v1/mcp/agents/$TARGET_ID/rpc" "$(json_body tools-call "$ALLOWED_TOOL")" "$AGENT_KEY" "$DATA_SCOPE_MISMATCH_RUN_ID" "$SUBJECT_ID"
expect_status 403 "deny data-scope mismatch tools/call"

request GET "/api/v1/audit/traces?runId=$DATA_SCOPE_MISMATCH_RUN_ID&decision=denied"
expect_status 200 "fetch data-scope denial trace"
assert_data_scope_denied_trace

request GET "/api/v1/tenants/$CHILD_TENANT_ID/access-profile?targetId=$TARGET_ID&capabilityId=$ALLOWED_CAPABILITY_ID&callerInstanceId=$CALLER_ID&traceLimit=10"
expect_status 200 "fetch data-scope mismatch access profile"
assert_data_scope_mismatch_profile

echo "core journey scenario complete"
