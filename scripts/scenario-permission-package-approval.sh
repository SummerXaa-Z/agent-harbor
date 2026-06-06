#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:9090}"
ADMIN_KEY="${ADMIN_KEY:-}"
RUN_ID="${RUN_ID:-permission-package-approval-$(date +%Y%m%d%H%M%S)}"
ROOT_TENANT_ID="${ROOT_TENANT_ID:-tenant-root-${RUN_ID}}"
CHILD_TENANT_ID="${CHILD_TENANT_ID:-tenant-child-${RUN_ID}}"
GRANDCHILD_TENANT_ID="${GRANDCHILD_TENANT_ID:-tenant-grandchild-${RUN_ID}}"
WORKSPACE_ID="${WORKSPACE_ID:-ws-permission-package-approval}"
MOCK_MCP_HOST="${MOCK_MCP_HOST:-127.0.0.1}"
MOCK_MCP_PORT="${MOCK_MCP_PORT:-8787}"
MCP_ENDPOINT="${MCP_ENDPOINT:-http://${MOCK_MCP_HOST}:${MOCK_MCP_PORT}/mcp}"
START_MOCK_MCP="${START_MOCK_MCP:-true}"
READ_TOOL="${READ_TOOL:-search_customer}"
WRITE_TOOL="${WRITE_TOOL:-update_ticket}"
DENIED_TOOL="${DENIED_TOOL:-export_contracts}"
REGION="${REGION:-us-east}"
REQUEST_TEXT="${REQUEST_TEXT:-Allow support triage reads and bounded ticket updates for this tenant.}"
TEMPLATE_ID="${TEMPLATE_ID:-support-ticket-triage}"
SUBJECT_SELECTOR="${SUBJECT_SELECTOR:-user:support-*}"
SUBJECT_ID="${SUBJECT_ID:-user:support-001}"

HTTP_STATUS=""
HTTP_BODY=""
MOCK_MCP_PID=""
CALLER_ID=""
TARGET_ID=""
READ_CAPABILITY_ID=""
WRITE_CAPABILITY_ID=""
DENIED_CAPABILITY_ID=""
APPROVAL_REQUEST_ID=""
APPLICATION_ID=""

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
      echo "local MCP endpoints require: AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true make run" >&2
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
elif kind == "permission-package":
    caller_id, region, request_text, subject_selector, target_id, template_id, tenant_id, workspace_id = sys.argv[2:10]
    body = {
        "callerInstanceId": caller_id,
        "region": region,
        "requestText": request_text,
        "subjectSelector": subject_selector,
        "targetId": target_id,
        "templateId": template_id,
        "tenantId": tenant_id,
        "workspaceId": workspace_id,
    }
    if len(sys.argv) > 10 and sys.argv[10]:
        body["approvalRequestId"] = sys.argv[10]
elif kind == "approval-resolution":
    reviewer, comment = sys.argv[2], sys.argv[3]
    body = {"reviewer": reviewer, "comment": comment}
elif kind == "tools-list":
    body = {"jsonrpc": "2.0", "id": "tools-list", "method": "tools/list"}
elif kind == "tools-call":
    tool = sys.argv[2]
    body = {
        "jsonrpc": "2.0",
        "id": f"call-{tool}",
        "method": "tools/call",
        "params": {
            "name": tool,
            "arguments": {
                "ticketId": "T-1000",
                "status": "triaged",
                "query": "acme",
            },
        },
    }
else:
    raise SystemExit(f"unknown body kind: {kind}")
print(json.dumps(body, separators=(",", ":")))
PY
}

permission_package_body() {
  local approval_request_id="${1:-}"
  json_body permission-package "$CALLER_ID" "$REGION" "$REQUEST_TEXT" "$SUBJECT_SELECTOR" "$TARGET_ID" "$TEMPLATE_ID" "$CHILD_TENANT_ID" "$WORKSPACE_ID" "$approval_request_id"
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

assert_discovered_capabilities() {
  RESPONSE_BODY="$HTTP_BODY" python3 - "$READ_TOOL" "$WRITE_TOOL" "$DENIED_TOOL" <<'PY'
import json
import os
import sys

doc = json.loads(os.environ["RESPONSE_BODY"])
read_tool, write_tool, denied_tool = sys.argv[1:4]
by_key = {capability["key"]: capability for capability in doc["data"]}
expected = {
    read_tool: "read",
    write_tool: "write",
    denied_tool: "export",
}
for key, action in expected.items():
    if key not in by_key:
        raise SystemExit(f"missing discovered capability {key!r}; got {sorted(by_key)}")
    if by_key[key].get("action") != action:
        raise SystemExit(f"capability {key!r} action={by_key[key].get('action')!r} want {action!r}")
print("support MCP capability discovery verified")
PY
}

assert_approval_required_draft() {
  RESPONSE_BODY="$HTTP_BODY" python3 - "$READ_TOOL" "$WRITE_TOOL" "$DENIED_TOOL" <<'PY'
import json
import os
import sys

draft = json.loads(os.environ["RESPONSE_BODY"])["data"]
read_tool, write_tool, denied_tool = sys.argv[1:4]
if draft["template"]["id"] != "support-ticket-triage":
    raise SystemExit(f"unexpected template: {draft['template']}")
if draft["readiness"]["canApply"] is not True:
    raise SystemExit(f"draft should be ready: {draft['readiness']}")
policy = draft["policyGate"]
if policy["decision"] != "approval_required" or policy["canApplyDirectly"] is not False:
    raise SystemExit(f"draft should require approval: {policy}")
if not policy.get("reasons"):
    raise SystemExit(f"approval-required draft should include policy reasons: {policy}")
allowed = {capability["key"] for capability in draft["allowedCapabilities"]}
blocked = {capability["key"] for capability in draft["blockedCapabilities"]}
for key in (read_tool, write_tool):
    if key not in allowed:
        raise SystemExit(f"allowed capability {key!r} missing: {allowed}")
if denied_tool not in blocked:
    raise SystemExit(f"blocked capability {denied_tool!r} missing: {blocked}")
if len(draft.get("dataScopes", [])) != 1:
    raise SystemExit(f"expected one package data scope: {draft.get('dataScopes')}")
scope = draft["dataScopes"][0]
if scope.get("dataDomain") != "support" or scope.get("region") != "us-east":
    raise SystemExit(f"unexpected package data scope: {scope}")
print("approval-required permission package draft verified")
PY
}

assert_approval_request() {
  local expected_status="$1"
  local expected_id="${2:-}"
  RESPONSE_BODY="$HTTP_BODY" python3 - "$expected_status" "$expected_id" "$WRITE_TOOL" <<'PY'
import json
import os
import sys

approval = json.loads(os.environ["RESPONSE_BODY"])["data"]
expected_status, expected_id, write_tool = sys.argv[1:4]
if expected_id and approval["id"] != expected_id:
    raise SystemExit(f"approval id={approval['id']!r} want {expected_id!r}")
if approval["status"] != expected_status:
    raise SystemExit(f"approval status={approval['status']!r} want {expected_status!r}")
if approval["templateId"] != "support-ticket-triage" or approval["templateVersion"] != 1:
    raise SystemExit(f"unexpected approval template: {approval}")
if write_tool not in approval.get("allowedCapabilityKeys", []):
    raise SystemExit(f"approval missing write tool {write_tool!r}: {approval.get('allowedCapabilityKeys')}")
if approval["policyGate"]["decision"] != "approval_required":
    raise SystemExit(f"approval missing policy gate snapshot: {approval['policyGate']}")
if not approval.get("expiresAt"):
    raise SystemExit(f"approval request should include expiresAt: {approval}")
print(f"approval request {expected_status} verified")
PY
}

assert_listed_approval() {
  local expected_id="$1"
  RESPONSE_BODY="$HTTP_BODY" python3 - "$expected_id" <<'PY'
import json
import os
import sys

rows = json.loads(os.environ["RESPONSE_BODY"])["data"]
expected_id = sys.argv[1]
if not rows:
    raise SystemExit("expected one listed approval request")
if rows[0]["id"] != expected_id or rows[0]["status"] != "pending":
    raise SystemExit(f"unexpected listed approval request: {rows[:1]}")
print("approval request list verified")
PY
}

assert_consumed_approval() {
  RESPONSE_BODY="$HTTP_BODY" python3 - "$APPROVAL_REQUEST_ID" "$APPLICATION_ID" <<'PY'
import json
import os
import sys

rows = json.loads(os.environ["RESPONSE_BODY"])["data"]
approval_request_id, application_id = sys.argv[1:3]
if not rows:
    raise SystemExit("expected one listed consumed approval request")
approval = rows[0]
if approval["id"] != approval_request_id or approval["status"] != "approved":
    raise SystemExit(f"unexpected consumed approval request: {approval}")
if approval.get("consumedByApplicationId") != application_id:
    raise SystemExit(f"approval request was not consumed by application {application_id!r}: {approval}")
if not approval.get("consumedAt"):
    raise SystemExit(f"approval request missing consumedAt: {approval}")
print("approval request consumption verified")
PY
}

assert_body_contains() {
  local expected="$1"
  local label="$2"
  if [[ "$HTTP_BODY" != *"$expected"* ]]; then
    echo "expected $label body to contain $expected" >&2
    echo "$HTTP_BODY" >&2
    exit 1
  fi
}

assert_apply_response() {
  local approval_request_id="$1"
  RESPONSE_BODY="$HTTP_BODY" python3 - "$approval_request_id" "$READ_TOOL" "$WRITE_TOOL" <<'PY'
import json
import os
import sys

doc = json.loads(os.environ["RESPONSE_BODY"])["data"]
approval_request_id, read_tool, write_tool = sys.argv[1:4]
if doc["draft"]["policyGate"]["decision"] != "approval_required":
    raise SystemExit(f"applied draft should preserve policy gate: {doc['draft']['policyGate']}")
application = doc.get("application")
if not application:
    raise SystemExit(f"apply response missing application: {doc}")
if application["templateId"] != "support-ticket-triage" or application["templateVersion"] != 1:
    raise SystemExit(f"unexpected application template: {application}")
allowed_keys = set(application.get("allowedCapabilityKeys", []))
for key in (read_tool, write_tool):
    if key not in allowed_keys:
        raise SystemExit(f"application missing allowed key {key!r}: {allowed_keys}")
if len(doc.get("tenantEntitlements", [])) < 2 or len(doc.get("workspaceAssignments", [])) < 2 or len(doc.get("instanceAssignments", [])) < 2:
    raise SystemExit(f"expected at least two assignments for read/write support tools: {doc}")
if not approval_request_id:
    raise SystemExit("approval request id must be provided to apply response assertion")
print(application["id"])
PY
}

assert_tools_list_filtered() {
  RESPONSE_BODY="$HTTP_BODY" python3 - "$READ_TOOL" "$WRITE_TOOL" "$DENIED_TOOL" <<'PY'
import json
import os
import sys

doc = json.loads(os.environ["RESPONSE_BODY"])
read_tool, write_tool, denied_tool = sys.argv[1:4]
tools = {tool["name"] for tool in doc["result"]["tools"]}
for key in (read_tool, write_tool):
    if key not in tools:
        raise SystemExit(f"allowed tool {key!r} missing from tools/list: {tools}")
if denied_tool in tools:
    raise SystemExit(f"denied tool {denied_tool!r} should have been filtered: {tools}")
print("tools/list filtering verified")
PY
}

assert_trace_decision() {
  local expected_decision="$1"
  local expected_capability_id="${2:-}"
  RESPONSE_BODY="$HTTP_BODY" python3 - "$expected_decision" "$expected_capability_id" <<'PY'
import json
import os
import sys

rows = json.loads(os.environ["RESPONSE_BODY"])["data"]
expected_decision, expected_capability_id = sys.argv[1:3]
if not rows:
    raise SystemExit("expected at least one trace")
trace = rows[0]
if trace.get("decision") != expected_decision:
    raise SystemExit(f"trace decision={trace.get('decision')!r} want {expected_decision!r}; trace={trace}")
if expected_capability_id and trace.get("capabilityId") != expected_capability_id:
    raise SystemExit(f"trace capabilityId={trace.get('capabilityId')!r} want {expected_capability_id!r}; trace={trace}")
print(f"{expected_decision} trace evidence verified")
PY
}

assert_profile_chain() {
  RESPONSE_BODY="$HTTP_BODY" python3 - "$CHILD_TENANT_ID" "$GRANDCHILD_TENANT_ID" "$TARGET_ID" "$READ_CAPABILITY_ID" "$WRITE_CAPABILITY_ID" <<'PY'
import json
import os
import sys

child_id, grandchild_id, target_id, read_capability_id, write_capability_id = sys.argv[1:6]
profile = json.loads(os.environ["RESPONSE_BODY"])["data"]
if profile["tenant"]["id"] != child_id:
    raise SystemExit(f"tenant id mismatch: {profile['tenant']}")
scope_ids = [row["id"] for row in profile["scopeTenants"]]
if scope_ids != [child_id, grandchild_id]:
    raise SystemExit(f"scope tenants = {scope_ids}")
summary = profile["summary"]
expected_counts = {
    "grantCount": 2,
    "targetCount": 1,
    "capabilityCount": 2,
    "workspaceAssignmentCount": 2,
    "instanceAssignmentCount": 2,
}
for key, expected in expected_counts.items():
    if summary.get(key) != expected:
        raise SystemExit(f"summary[{key}]={summary.get(key)} want {expected}; summary={summary}")
if summary.get("recentAllowedTraceCount", 0) < 1 or summary.get("recentDeniedTraceCount", 0) < 1:
    raise SystemExit(f"expected allowed and denied trace evidence, summary={summary}")
capability_ids = set()
for grant in profile["grants"]:
    if grant.get("scopeStatus") != "valid":
        raise SystemExit(f"grant not valid: {grant}")
    if grant["target"]["id"] != target_id:
        raise SystemExit(f"target mismatch: {grant}")
    capability_ids.add(grant["capability"]["id"])
    workspace = grant["workspaceAssignments"][0]
    instance = workspace["instanceAssignments"][0]
    scope = instance["effectiveInstanceDataScopes"][0]
    expected_scope = {
        "dataDomain": "support",
        "region": "us-east",
        "tenantFilter": f"tenant_id = '{child_id}'",
    }
    for key, expected in expected_scope.items():
        if scope.get(key) != expected:
            raise SystemExit(f"scope[{key}]={scope.get(key)!r} want {expected!r}; scope={scope}")
expected_capabilities = {read_capability_id, write_capability_id}
if capability_ids != expected_capabilities:
    raise SystemExit(f"profile capabilities={capability_ids} want {expected_capabilities}")
print("tenant access profile approval grant chain verified")
PY
}

assert_application_list() {
  RESPONSE_BODY="$HTTP_BODY" python3 - "$APPLICATION_ID" "$WRITE_TOOL" <<'PY'
import json
import os
import sys

rows = json.loads(os.environ["RESPONSE_BODY"])["data"]
application_id, write_tool = sys.argv[1:3]
if not rows:
    raise SystemExit("expected one permission package application")
application = rows[0]
if application["id"] != application_id or application["templateId"] != "support-ticket-triage":
    raise SystemExit(f"unexpected permission package application: {application}")
if write_tool not in application.get("allowedCapabilityKeys", []):
    raise SystemExit(f"application missing write tool {write_tool!r}: {application}")
print("permission package application list verified")
PY
}

assert_application_impact() {
  RESPONSE_BODY="$HTTP_BODY" python3 - "$APPLICATION_ID" "$READ_TOOL" "$WRITE_TOOL" <<'PY'
import json
import os
import sys

impact = json.loads(os.environ["RESPONSE_BODY"])["data"]
application_id, read_tool, write_tool = sys.argv[1:4]
if impact["application"]["id"] != application_id:
    raise SystemExit(f"impact application id mismatch: {impact['application']}")
summary = impact["summary"]
expected = {
    "createdObjectCount": 6,
    "activeObjectCount": 6,
    "missingObjectCount": 0,
}
for key, value in expected.items():
    if summary.get(key) != value:
        raise SystemExit(f"impact summary[{key}]={summary.get(key)} want {value}; summary={summary}")
if summary.get("rollbackReady") is not True:
    raise SystemExit(f"impact should be rollback-review ready: {summary}")
object_types = {row["type"] for row in impact.get("createdObjects", [])}
for object_type in ("tenant_entitlement", "workspace_assignment", "instance_assignment"):
    if object_type not in object_types:
        raise SystemExit(f"impact missing object type {object_type!r}: {object_types}")
capability_keys = {row.get("key"): row for row in impact.get("capabilityReviews", [])}
for key in (read_tool, write_tool):
    row = capability_keys.get(key)
    if not row:
        raise SystemExit(f"impact missing capability review for {key!r}: {capability_keys}")
    if row.get("rollbackAction") != "manual_review":
        raise SystemExit(f"impact capability {key!r} rollbackAction={row.get('rollbackAction')!r}: {row}")
review = impact["rollbackReview"]
if review.get("ready") is not True:
    raise SystemExit(f"rollback review should be ready: {review}")
if review.get("blockers") != []:
    raise SystemExit(f"rollback blockers should be an empty array: {review}")
if not review.get("steps"):
    raise SystemExit(f"rollback review should include steps: {review}")
print("permission package application impact verified")
PY
}

assert_applied_audit_event() {
  RESPONSE_BODY="$HTTP_BODY" python3 - "$APPLICATION_ID" "$APPROVAL_REQUEST_ID" <<'PY'
import json
import os
import sys

rows = json.loads(os.environ["RESPONSE_BODY"])["data"]
application_id, approval_request_id = sys.argv[1:3]
if len(rows) != 1:
    raise SystemExit(f"expected exactly one applied audit event, got {len(rows)}: {rows}")
event = rows[0]
metadata = event.get("metadata") or {}
if event.get("resourceId") != application_id:
    raise SystemExit(f"audit event resourceId={event.get('resourceId')!r} want {application_id!r}: {event}")
if metadata.get("applicationId") != application_id:
    raise SystemExit(f"audit metadata applicationId mismatch: {metadata}")
if metadata.get("approvalRequestId") != approval_request_id:
    raise SystemExit(f"audit metadata approvalRequestId mismatch: {metadata}")
print("permission package approval audit evidence verified")
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

echo "AgentHarbor permission package approval scenario"
echo "BASE_URL=$BASE_URL"
echo "MCP_ENDPOINT=$MCP_ENDPOINT"
echo "RUN_ID=$RUN_ID"
echo "SUBJECT_ID=$SUBJECT_ID"
if [[ -n "$ADMIN_KEY" ]]; then
  echo "ADMIN_KEY=provided"
else
  echo "ADMIN_KEY=not set"
fi

start_mock_mcp

request GET "/healthz"
expect_status 200 "AgentHarbor health check"

request POST "/api/v1/tenants" "$(json_body tenant "$ROOT_TENANT_ID" "" "Permission Package Approval Root")"
expect_status 201 "create root tenant"
request POST "/api/v1/tenants" "$(json_body tenant "$CHILD_TENANT_ID" "$ROOT_TENANT_ID" "Permission Package Approval Team")"
expect_status 201 "create child tenant"
request POST "/api/v1/tenants" "$(json_body tenant "$GRANDCHILD_TENANT_ID" "$CHILD_TENANT_ID" "Permission Package Approval Project")"
expect_status 201 "create grandchild tenant"
echo "created tenant tree: $ROOT_TENANT_ID -> $CHILD_TENANT_ID -> $GRANDCHILD_TENANT_ID"

request POST "/api/v1/agents" "$(json_body agent "$CHILD_TENANT_ID" "$WORKSPACE_ID" "Permission Package Approval Caller" "local")"
expect_status 201 "create caller agent"
CALLER_ID="$(json_get data.id)"

request POST "/api/v1/agent-keys" "{\"agentId\":\"$CALLER_ID\",\"name\":\"permission package approval key\"}"
expect_status 201 "create caller key"
AGENT_KEY="$(json_get data.key)"

request POST "/api/v1/agents" "$(json_body agent "$ROOT_TENANT_ID" "$WORKSPACE_ID" "Permission Package Approval MCP Target" "mcp" "$MCP_ENDPOINT")"
expect_status 201 "create MCP target"
TARGET_ID="$(json_get data.id)"

request POST "/api/v1/targets/$TARGET_ID/capabilities:refresh"
expect_status 200 "discover MCP capabilities"
assert_discovered_capabilities
READ_CAPABILITY_ID="$(capability_id_for_key "$READ_TOOL")"
WRITE_CAPABILITY_ID="$(capability_id_for_key "$WRITE_TOOL")"
DENIED_CAPABILITY_ID="$(capability_id_for_key "$DENIED_TOOL")"
echo "discovered capabilities: read=$READ_CAPABILITY_ID write=$WRITE_CAPABILITY_ID deny=$DENIED_CAPABILITY_ID"

request POST "/api/v1/permission-packages/drafts" "$(permission_package_body)"
expect_status 200 "draft approval-required permission package"
assert_approval_required_draft

request POST "/api/v1/permission-packages:apply" "$(permission_package_body)"
expect_status 400 "reject approval-required package without approval"
echo "direct apply without approval rejected"

request POST "/api/v1/permission-packages/approval-requests" "$(permission_package_body)"
expect_status 201 "create approval request"
assert_approval_request "pending"
APPROVAL_REQUEST_ID="$(json_get data.id)"

request GET "/api/v1/permission-packages/approval-requests?tenantId=$ROOT_TENANT_ID&workspaceId=$WORKSPACE_ID&templateId=$TEMPLATE_ID&targetId=$TARGET_ID&callerInstanceId=$CALLER_ID&status=pending&limit=1"
expect_status 200 "list pending approval requests"
assert_listed_approval "$APPROVAL_REQUEST_ID"

request POST "/api/v1/permission-packages/approval-requests/$APPROVAL_REQUEST_ID/approve" "$(json_body approval-resolution "security-reviewer" "approved for local production journey scenario")"
expect_status 200 "approve approval request"
assert_approval_request "approved" "$APPROVAL_REQUEST_ID"

request POST "/api/v1/permission-packages:apply" "$(permission_package_body "$APPROVAL_REQUEST_ID")"
expect_status 201 "apply approved permission package"
APPLICATION_ID="$(assert_apply_response "$APPROVAL_REQUEST_ID")"
echo "applied permission package application: $APPLICATION_ID"

request GET "/api/v1/permission-packages/approval-requests?tenantId=$ROOT_TENANT_ID&workspaceId=$WORKSPACE_ID&templateId=$TEMPLATE_ID&targetId=$TARGET_ID&callerInstanceId=$CALLER_ID&status=approved&limit=1"
expect_status 200 "list consumed approval request"
assert_consumed_approval

request POST "/api/v1/permission-packages:apply" "$(permission_package_body "$APPROVAL_REQUEST_ID")"
expect_status 400 "reject consumed approval request"
assert_body_contains "already consumed" "consumed approval retry"
echo "consumed approval request reuse rejected"

request POST "/api/v1/mcp/agents/$TARGET_ID/rpc" "$(json_body tools-list)" "$AGENT_KEY" "$RUN_ID-tools-list" "$SUBJECT_ID"
expect_status 200 "filtered tools/list"
assert_tools_list_filtered

request POST "/api/v1/mcp/agents/$TARGET_ID/rpc" "$(json_body tools-call "$DENIED_TOOL")" "$AGENT_KEY" "$RUN_ID-denied" "$SUBJECT_ID"
expect_status 403 "deny blocked tools/call"
echo "denied blocked tool: $DENIED_TOOL"

request GET "/api/v1/audit/traces?runId=$RUN_ID-denied"
expect_status 200 "list denied trace"
assert_trace_decision "denied"

request POST "/api/v1/mcp/agents/$TARGET_ID/rpc" "$(json_body tools-call "$WRITE_TOOL")" "$AGENT_KEY" "$RUN_ID-allowed" "$SUBJECT_ID"
expect_2xx "allow approved tools/call"
echo "allowed approved tool: $WRITE_TOOL"

request GET "/api/v1/audit/traces?runId=$RUN_ID-allowed"
expect_status 200 "list allowed trace"
assert_trace_decision "allowed" "$WRITE_CAPABILITY_ID"

request GET "/api/v1/tenants/$CHILD_TENANT_ID/access-profile?traceLimit=10"
expect_status 200 "fetch tenant access profile"
assert_profile_chain

request GET "/api/v1/permission-packages/applications?tenantId=$ROOT_TENANT_ID&workspaceId=$WORKSPACE_ID&templateId=$TEMPLATE_ID&targetId=$TARGET_ID&callerInstanceId=$CALLER_ID&limit=1"
expect_status 200 "list permission package applications"
assert_application_list

request GET "/api/v1/permission-packages/applications/$APPLICATION_ID/impact?tenantId=$ROOT_TENANT_ID&workspaceId=$WORKSPACE_ID"
expect_status 200 "review permission package application impact"
assert_application_impact

request GET "/api/v1/audit/events?action=permission_package.applied&resourceId=$APPLICATION_ID&limit=1"
expect_status 200 "list applied audit events"
assert_applied_audit_event

echo "permission package approval scenario complete"
