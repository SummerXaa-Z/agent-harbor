#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:9090}"
ADMIN_KEY="${ADMIN_KEY:-}"
REQUESTER_ACTOR="${REQUESTER_ACTOR:-requester}"
APPROVAL_REVIEWER="${APPROVAL_REVIEWER:-security-reviewer}"
REQUESTER_ADMIN_KEY="${REQUESTER_ADMIN_KEY:-${ADMIN_KEY:-}}"
REVIEWER_ADMIN_KEY="${REVIEWER_ADMIN_KEY:-${ADMIN_KEY:-}}"
ADMIN_KEY="${ADMIN_KEY:-$REQUESTER_ADMIN_KEY}"
RUN_ID="${RUN_ID:-permission-package-approval-$(date +%Y%m%d%H%M%S)}"
ROOT_TENANT_ID="${ROOT_TENANT_ID:-tenant-root-${RUN_ID}}"
CHILD_TENANT_ID="${CHILD_TENANT_ID:-tenant-child-${RUN_ID}}"
GRANDCHILD_TENANT_ID="${GRANDCHILD_TENANT_ID:-tenant-grandchild-${RUN_ID}}"
WORKSPACE_ID="${WORKSPACE_ID:-ws-permission-package-approval}"
MOCK_MCP_HOST="${MOCK_MCP_HOST:-127.0.0.1}"
MOCK_MCP_PORT="${MOCK_MCP_PORT:-8787}"
MCP_ENDPOINT="${MCP_ENDPOINT:-http://${MOCK_MCP_HOST}:${MOCK_MCP_PORT}/mcp}"
START_MOCK_MCP="${START_MOCK_MCP:-true}"
MCP_SERVER_MODE="${MCP_SERVER_MODE:-mock}"
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
SYSTEM_INFO_BODY=""

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
  local admin_key="${7:-$ADMIN_KEY}"
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

  if [[ -n "$admin_key" ]]; then
    args+=(-H "X-Admin-Key: $admin_key")
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
elif kind == "mcp-explain-permission-package":
    caller_id, region, request_text, subject_selector, target_id, template_id, tenant_id, workspace_id = sys.argv[2:10]
    body = {
        "jsonrpc": "2.0",
        "id": "explain-permission-package",
        "method": "tools/call",
        "params": {
            "name": "explain_permission_package_draft",
            "arguments": {
                "callerInstanceId": caller_id,
                "region": region,
                "requestText": request_text,
                "subjectSelector": subject_selector,
                "targetId": target_id,
                "templateId": template_id,
                "tenantId": tenant_id,
                "workspaceId": workspace_id,
            },
        },
    }
elif kind == "mcp-explain-access-decision":
    caller_id, capability_id, subject_id, target_id, tenant_id, workspace_id = sys.argv[2:8]
    body = {
        "jsonrpc": "2.0",
        "id": "explain-access-decision",
        "method": "tools/call",
        "params": {
            "name": "explain_access_decision",
            "arguments": {
                "callerInstanceId": caller_id,
                "capabilityId": capability_id,
                "subjectId": subject_id,
                "targetId": target_id,
                "tenantId": tenant_id,
                "workspaceId": workspace_id,
            },
        },
    }
elif kind == "mcp-permission-package-write":
    tool, confirmation_mode, caller_id, region, request_text, subject_selector, target_id, template_id, tenant_id, workspace_id = sys.argv[2:12]
    arguments = {
        "callerInstanceId": caller_id,
        "region": region,
        "requestText": request_text,
        "subjectSelector": subject_selector,
        "targetId": target_id,
        "templateId": template_id,
        "tenantId": tenant_id,
        "workspaceId": workspace_id,
    }
    if len(sys.argv) > 12 and sys.argv[12]:
        arguments["approvalRequestId"] = sys.argv[12]
    if confirmation_mode == "confirmed":
        arguments["confirmation"] = {
            "confirmed": True,
            "reason": f"Scenario confirmed Management MCP write for {tool}.",
        }
    elif confirmation_mode != "missing":
        raise SystemExit(f"unknown confirmation mode: {confirmation_mode}")
    body = {
        "jsonrpc": "2.0",
        "id": f"call-{tool}",
        "method": "tools/call",
        "params": {"name": tool, "arguments": arguments},
    }
elif kind == "mcp-approval-resolution-write":
    tool, approval_request_id, reviewer, comment, confirmation_mode = sys.argv[2:7]
    arguments = {"id": approval_request_id, "reviewer": reviewer, "comment": comment}
    if confirmation_mode == "confirmed":
        arguments["confirmation"] = {
            "confirmed": True,
            "reason": f"Scenario confirmed Management MCP write for {tool}.",
        }
    elif confirmation_mode != "missing":
        raise SystemExit(f"unknown confirmation mode: {confirmation_mode}")
    body = {
        "jsonrpc": "2.0",
        "id": f"call-{tool}",
        "method": "tools/call",
        "params": {"name": tool, "arguments": arguments},
    }
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

mcp_explain_permission_package_body() {
  json_body mcp-explain-permission-package "$CALLER_ID" "$REGION" "$REQUEST_TEXT" "$SUBJECT_SELECTOR" "$TARGET_ID" "$TEMPLATE_ID" "$CHILD_TENANT_ID" "$WORKSPACE_ID"
}

mcp_explain_access_decision_body() {
  local capability_id="$1"
  json_body mcp-explain-access-decision "$CALLER_ID" "$capability_id" "$SUBJECT_ID" "$TARGET_ID" "$CHILD_TENANT_ID" "$WORKSPACE_ID"
}

mcp_permission_package_write_body() {
  local tool="$1"
  local confirmation="${2:-confirmed}"
  local approval_request_id="${3:-}"
  json_body mcp-permission-package-write "$tool" "$confirmation" "$CALLER_ID" "$REGION" "$REQUEST_TEXT" "$SUBJECT_SELECTOR" "$TARGET_ID" "$TEMPLATE_ID" "$CHILD_TENANT_ID" "$WORKSPACE_ID" "$approval_request_id"
}

mcp_approval_resolution_write_body() {
  local tool="$1"
  local approval_request_id="$2"
  local reviewer="$3"
  local comment="$4"
  local confirmation="${5:-confirmed}"
  json_body mcp-approval-resolution-write "$tool" "$approval_request_id" "$reviewer" "$comment" "$confirmation"
}

production_readiness_path() {
  local approval_request_id="${1:-}"
  python3 - "$approval_request_id" "$CALLER_ID" "$REGION" "$REQUEST_TEXT" "$SUBJECT_ID" "$SUBJECT_SELECTOR" "$TARGET_ID" "$TEMPLATE_ID" "$CHILD_TENANT_ID" "$WORKSPACE_ID" <<'PY'
import sys
from urllib.parse import urlencode

approval_request_id, caller_id, region, request_text, subject_id, subject_selector, target_id, template_id, tenant_id, workspace_id = sys.argv[1:11]
params = {
    "callerInstanceId": caller_id,
    "region": region,
    "requestText": request_text,
    "subjectId": subject_id,
    "subjectSelector": subject_selector,
    "targetId": target_id,
    "templateId": template_id,
    "tenantId": tenant_id,
    "traceLimit": "20",
    "workspaceId": workspace_id,
}
if approval_request_id:
    params["approvalRequestId"] = approval_request_id
print("/api/v1/permission-packages/production-readiness?" + urlencode(params))
PY
}

production_evidence_report_path() {
  local approval_request_id="${1:-}"
  production_readiness_path "$approval_request_id" | sed 's#/production-readiness?#/production-readiness/report?#'
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
next_action_codes = policy.get("nextActionCodes") or []
if "create_approval_request" not in next_action_codes:
    raise SystemExit(
        f"approval-required draft missing policy gate nextActionCodes={next_action_codes!r}: {policy}"
    )
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

assert_mcp_permission_package_explain() {
  RESPONSE_BODY="$HTTP_BODY" python3 <<'PY'
import json
import os

doc = json.loads(os.environ["RESPONSE_BODY"])
structured = doc.get("result", {}).get("structuredContent")
if not isinstance(structured, dict):
    raise SystemExit(f"missing structuredContent: {doc}")
if structured.get("outcome") != "approval_required":
    raise SystemExit(f"expected approval_required explanation: {structured}")
next_action_codes = structured.get("nextActionCodes") or []
if "create_approval_request" not in next_action_codes:
    raise SystemExit(
        f"MCP explanation nextActionCodes={next_action_codes!r} missing create_approval_request: {structured}"
    )
if "apply_permission_package" in next_action_codes:
    raise SystemExit(f"MCP explanation should not suggest direct apply code: {structured}")
policy_codes = structured.get("policyGate", {}).get("nextActionCodes") or []
if "create_approval_request" not in policy_codes:
    raise SystemExit(f"MCP explanation policy gate codes missing create_approval_request: {structured}")
print("management MCP permission package explanation verified")
PY
}

assert_mcp_access_explain() {
  local expected_outcome="$1"
  local expected_code="$2"
  RESPONSE_BODY="$HTTP_BODY" python3 - "$expected_outcome" "$expected_code" <<'PY'
import json
import os
import sys

expected_outcome, expected_code = sys.argv[1:3]
doc = json.loads(os.environ["RESPONSE_BODY"])
structured = doc.get("result", {}).get("structuredContent")
if not isinstance(structured, dict):
    raise SystemExit(f"missing structuredContent: {doc}")
if structured.get("outcome") != expected_outcome:
    raise SystemExit(f"expected {expected_outcome!r} access explanation: {structured}")
next_action_codes = structured.get("nextActionCodes") or []
if expected_code not in next_action_codes:
    raise SystemExit(
        f"access explanation nextActionCodes={next_action_codes!r} missing {expected_code!r}: {structured}"
    )
if expected_outcome == "denied" and "no_change_required" in next_action_codes:
    raise SystemExit(f"denied access explanation should not suggest no-change code: {structured}")
if expected_outcome == "allowed":
    decision = structured.get("decision") or {}
    if decision.get("allowed") is not True:
        raise SystemExit(f"allowed access explanation should include allowed decision: {structured}")
    if not structured.get("dataScopes"):
        raise SystemExit(f"allowed access explanation should include effective data scopes: {structured}")
print(f"management MCP access explanation verified: {expected_outcome}")
PY
}

assert_mcp_tool_error() {
  local expected_app_code="$1"
  local expected_text="$2"
  RESPONSE_BODY="$HTTP_BODY" python3 - "$expected_app_code" "$expected_text" <<'PY'
import json
import os
import sys

expected_app_code, expected_text = sys.argv[1:3]
doc = json.loads(os.environ["RESPONSE_BODY"])
error = doc.get("error") or {}
data = error.get("data") or {}
if doc.get("result") is not None:
    raise SystemExit(f"expected JSON-RPC error, got result: {doc}")
if data.get("appCode") != expected_app_code:
    raise SystemExit(f"JSON-RPC appCode={data.get('appCode')!r} want {expected_app_code!r}: {doc}")
if expected_text not in error.get("message", ""):
    raise SystemExit(f"JSON-RPC message missing {expected_text!r}: {doc}")
print(f"management MCP tool error verified: {expected_app_code}")
PY
}

assert_mcp_approval_request() {
  local expected_status="$1"
  local expected_id="${2:-}"
  RESPONSE_BODY="$HTTP_BODY" python3 - "$expected_status" "$expected_id" "$WRITE_TOOL" <<'PY'
import json
import os
import sys

expected_status, expected_id, write_tool = sys.argv[1:4]
doc = json.loads(os.environ["RESPONSE_BODY"])
approval = (doc.get("result") or {}).get("structuredContent")
if not isinstance(approval, dict):
    raise SystemExit(f"missing structuredContent approval request: {doc}")
if expected_id and approval.get("id") != expected_id:
    raise SystemExit(f"approval id={approval.get('id')!r} want {expected_id!r}")
if approval.get("status") != expected_status:
    raise SystemExit(f"approval status={approval.get('status')!r} want {expected_status!r}: {approval}")
if approval.get("templateId") != "support-ticket-triage" or approval.get("templateVersion") != 1:
    raise SystemExit(f"unexpected approval template: {approval}")
if write_tool not in approval.get("allowedCapabilityKeys", []):
    raise SystemExit(f"approval missing write tool {write_tool!r}: {approval.get('allowedCapabilityKeys')}")
if (approval.get("policyGate") or {}).get("decision") != "approval_required":
    raise SystemExit(f"approval missing policy gate snapshot: {approval.get('policyGate')}")
if not approval.get("expiresAt"):
    raise SystemExit(f"approval request should include expiresAt: {approval}")
print(f"management MCP approval request {expected_status} verified")
PY
}

assert_management_mcp_tool_safety() {
  RESPONSE_BODY="$HTTP_BODY" SYSTEM_INFO_BODY="$SYSTEM_INFO_BODY" python3 <<'PY'
import json
import os
import re

doc = json.loads(os.environ["RESPONSE_BODY"])
result = doc.get("result", {})
if result.get("metadataVersion") != 4:
    raise SystemExit(f"management MCP tools/list metadataVersion={result.get('metadataVersion')!r} want 4")
tools_by_name = {
    tool.get("name"): tool
    for tool in result.get("tools", [])
}

def required_contains(schema, field):
    required = schema.get("required") if isinstance(schema, dict) else None
    return isinstance(required, list) and field in required

def assert_confirmation_schema(name, input_schema):
    if not isinstance(input_schema, dict):
        raise SystemExit(f"management MCP tool {name!r} inputSchema should be an object: {input_schema!r}")
    properties = input_schema.get("properties")
    if not isinstance(properties, dict):
        raise SystemExit(f"management MCP tool {name!r} inputSchema should expose properties: {input_schema!r}")
    confirmation = properties.get("confirmation")
    if not isinstance(confirmation, dict) or confirmation.get("type") != "object":
        raise SystemExit(f"management MCP tool {name!r} input schema should expose confirmation object: {input_schema!r}")
    if not required_contains(input_schema, "confirmation"):
        raise SystemExit(f"management MCP tool {name!r} input schema should require confirmation: {input_schema!r}")
    confirmation_properties = confirmation.get("properties")
    if not isinstance(confirmation_properties, dict):
        raise SystemExit(f"management MCP tool {name!r} confirmation schema should expose properties: {confirmation!r}")
    confirmed = confirmation_properties.get("confirmed")
    if not isinstance(confirmed, dict) or confirmed.get("type") != "boolean":
        raise SystemExit(f"management MCP tool {name!r} confirmation schema should expose confirmed boolean: {confirmation!r}")
    reason = confirmation_properties.get("reason")
    if not isinstance(reason, dict) or reason.get("type") != "string" or reason.get("maxLength") != 500:
        raise SystemExit(f"management MCP tool {name!r} confirmation schema should expose reason string maxLength 500: {confirmation!r}")
    if not required_contains(confirmation, "confirmed") or not required_contains(confirmation, "reason"):
        raise SystemExit(f"management MCP tool {name!r} confirmation schema should require confirmed and reason: {confirmation!r}")

def tool_requires_confirmation(tool):
    safety = tool.get("safety") or {}
    execution = tool.get("execution") or {}
    return (
        execution.get("confirmationRequired") is True
        or safety.get("mutatesState") is True
        or safety.get("operationType") == "write"
    )

def tool_has_confirmation_schema(tool):
    input_schema = tool.get("inputSchema")
    if not isinstance(input_schema, dict):
        return False
    properties = input_schema.get("properties")
    if not isinstance(properties, dict):
        return False
    confirmation = properties.get("confirmation")
    if not isinstance(confirmation, dict) or confirmation.get("type") != "object":
        return False
    confirmation_properties = confirmation.get("properties")
    if not isinstance(confirmation_properties, dict):
        return False
    confirmed = confirmation_properties.get("confirmed")
    reason = confirmation_properties.get("reason")
    return (
        required_contains(input_schema, "confirmation")
        and required_contains(confirmation, "confirmed")
        and required_contains(confirmation, "reason")
        and isinstance(confirmed, dict)
        and confirmed.get("type") == "boolean"
        and isinstance(reason, dict)
        and reason.get("type") == "string"
        and reason.get("minLength") == 1
        and reason.get("maxLength") == 500
    )

system_info_body = os.environ.get("SYSTEM_INFO_BODY", "")
if system_info_body:
    system_info = json.loads(system_info_body).get("data", {})
    catalog = system_info.get("managementMcpToolCatalog") or {}
    if catalog.get("metadataVersion") != result.get("metadataVersion"):
        raise SystemExit(f"system info metadataVersion={catalog.get('metadataVersion')!r} does not match tools/list {result.get('metadataVersion')!r}")
    if not re.fullmatch(r"[a-f0-9]{64}", str(catalog.get("catalogDigest", ""))):
        raise SystemExit(f"system info catalogDigest should be a sha256 hex digest: {catalog}")
    if catalog.get("catalogDigest") != result.get("catalogDigest"):
        raise SystemExit(f"system info catalogDigest={catalog.get('catalogDigest')!r} does not match tools/list {result.get('catalogDigest')!r}")
    required_metadata = catalog.get("requiredMetadata")
    if required_metadata != ["safety", "access", "lifecycle", "execution"]:
        raise SystemExit(f"system info requiredMetadata={required_metadata!r} is not the v4 contract")
    confirmation_required = sum(1 for tool in tools_by_name.values() if tool_requires_confirmation(tool))
    with_confirmation_schema = sum(1 for tool in tools_by_name.values() if tool_requires_confirmation(tool) and tool_has_confirmation_schema(tool))
    expected_summary = {
        "toolCount": len(tools_by_name),
        "confirmationRequiredTools": confirmation_required,
        "toolsWithConfirmationSchema": with_confirmation_schema,
    }
    for key, expected in expected_summary.items():
        if catalog.get(key) != expected:
            raise SystemExit(f"system info management MCP catalog {key}={catalog.get(key)!r} want {expected!r}: {catalog}")

required_safety = {
    "explain_access_decision": {"operationType": "read", "readOnly": True, "mutatesState": False, "approvalMode": "none"},
    "draft_permission_package": {"operationType": "preview", "readOnly": True, "mutatesState": False, "approvalMode": "none"},
    "apply_permission_package": {"operationType": "write", "readOnly": False, "mutatesState": True, "approvalMode": "conditional"},
    "approve_permission_package_approval_request": {"operationType": "write", "readOnly": False, "mutatesState": True, "approvalMode": "reviewer"},
}
required_access = {
    "list_admin_identities": {"requiredRole": "platform_admin", "scopeBoundary": "platform", "reviewerBound": False},
    "draft_permission_package": {"requiredRole": "authenticated_admin", "scopeBoundary": "requested_scope", "reviewerBound": False},
    "approve_permission_package_approval_request": {"requiredRole": "authenticated_admin", "scopeBoundary": "reviewer_route", "reviewerBound": True},
}
required_execution = {
    "explain_access_decision": {"idempotency": "safe_repeat", "confirmationRequired": False},
    "create_admin_identity": {
        "idempotency": "not_idempotent",
        "confirmationRequired": True,
        "auditResourceType": "admin_identity",
        "returnsSecret": True,
    },
    "rotate_admin_identity_key": {
        "idempotency": "not_idempotent",
        "confirmationRequired": True,
        "auditResourceType": "admin_identity",
        "returnsSecret": True,
    },
    "apply_permission_package": {
        "idempotency": "conditional_repeat",
        "confirmationRequired": True,
        "preflightTool": "preflight_permission_package",
        "auditResourceType": "permission_package_application",
    },
    "approve_permission_package_approval_request": {
        "idempotency": "conditional_repeat",
        "confirmationRequired": True,
        "auditResourceType": "permission_package_approval_request",
    },
}
for name, expected in required_safety.items():
    safety = (tools_by_name.get(name) or {}).get("safety")
    if safety != expected:
        raise SystemExit(f"management MCP tool {name!r} safety={safety!r} want {expected!r}")
for name, expected in required_access.items():
    access = (tools_by_name.get(name) or {}).get("access")
    if access != expected:
        raise SystemExit(f"management MCP tool {name!r} access={access!r} want {expected!r}")
for name, expected in required_execution.items():
    execution = (tools_by_name.get(name) or {}).get("execution")
    if execution != expected:
        raise SystemExit(f"management MCP tool {name!r} execution={execution!r} want {expected!r}")
legacy_lifecycle = (tools_by_name.get("export_permission_package_production_evidence") or {}).get("lifecycle")
expected_legacy_lifecycle = {
    "status": "compatibility_alias",
    "preferredName": "export_permission_package_production_report",
}
if legacy_lifecycle != expected_legacy_lifecycle:
    raise SystemExit(
        f"management MCP legacy report tool lifecycle={legacy_lifecycle!r} want {expected_legacy_lifecycle!r}"
    )
for name, tool in tools_by_name.items():
    safety = tool.get("safety") or {}
    access = tool.get("access") or {}
    lifecycle = tool.get("lifecycle") or {}
    execution = tool.get("execution") or {}
    if safety.get("operationType") in ("", "unspecified", None) or safety.get("approvalMode") in ("", "unspecified", None):
        raise SystemExit(f"management MCP tool {name!r} has incomplete safety metadata: {safety!r}")
    if access.get("requiredRole") in ("", "unspecified", None) or access.get("scopeBoundary") in ("", "unspecified", None):
        raise SystemExit(f"management MCP tool {name!r} has incomplete access metadata: {access!r}")
    if lifecycle.get("status") in ("", "unspecified", None):
        raise SystemExit(f"management MCP tool {name!r} has incomplete lifecycle metadata: {lifecycle!r}")
    if lifecycle.get("status") == "compatibility_alias" and not lifecycle.get("preferredName"):
        raise SystemExit(f"management MCP tool {name!r} compatibility alias is missing preferredName: {lifecycle!r}")
    if execution.get("idempotency") in ("", "unspecified", None):
        raise SystemExit(f"management MCP tool {name!r} has incomplete execution metadata: {execution!r}")
    if execution.get("confirmationRequired") not in (True, False):
        raise SystemExit(f"management MCP tool {name!r} execution missing confirmationRequired: {execution!r}")
    if safety.get("mutatesState") is True and execution.get("confirmationRequired") is not True:
        raise SystemExit(f"management MCP write tool {name!r} must require confirmation: {execution!r}")
    if execution.get("idempotency") == "not_idempotent" and not execution.get("confirmationRequired"):
        raise SystemExit(f"management MCP tool {name!r} non-idempotent execution must require confirmation: {execution!r}")
    input_schema = tool.get("inputSchema")
    input_properties = input_schema.get("properties") if isinstance(input_schema, dict) else {}
    requires_confirmation = tool_requires_confirmation(tool)
    if requires_confirmation:
        assert_confirmation_schema(name, input_schema)
    elif isinstance(input_properties, dict) and "confirmation" in input_properties:
        raise SystemExit(f"management MCP read tool {name!r} should not expose confirmation schema: {input_schema!r}")
print("management MCP tool catalog metadata and confirmation schemas verified")
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

assert_permission_package_preflight() {
  local expected_can_apply="$1"
  local expected_check="$2"
  RESPONSE_BODY="$HTTP_BODY" python3 - "$expected_can_apply" "$expected_check" <<'PY'
import json
import os
import sys

preflight = json.loads(os.environ["RESPONSE_BODY"])["data"]
expected_can_apply = sys.argv[1] == "true"
expected_check = sys.argv[2]
summary = preflight.get("summary") or {}
if summary.get("canApply") is not expected_can_apply:
    raise SystemExit(f"preflight canApply={summary.get('canApply')!r} want {expected_can_apply!r}: {summary}")
codes = {check.get("code"): check for check in preflight.get("checks", [])}
if expected_check not in codes:
    raise SystemExit(f"preflight missing check {expected_check!r}: {codes}")
expected_next_actions = {
    "approval_request_missing": "create_approval_request",
    "approval_request_ready": "apply_permission_package",
    "application_already_applied": "review_current_application",
}
expected_next_action = expected_next_actions.get(expected_check)
next_action_codes = preflight.get("nextActionCodes") or []
if expected_next_action and expected_next_action not in next_action_codes:
    raise SystemExit(
        f"preflight nextActionCodes={next_action_codes!r} missing {expected_next_action!r}: {preflight}"
    )
if expected_can_apply:
    if summary.get("blockingCount") != 0:
        raise SystemExit(f"ready preflight should have zero blockers: {summary}")
    if summary.get("plannedCapabilityCount", 0) < 1:
        raise SystemExit(f"ready preflight should include planned capabilities: {summary}")
else:
    if summary.get("blockingCount", 0) < 1:
        raise SystemExit(f"blocked preflight should include blockers: {summary}")
if "application" in preflight:
    raise SystemExit(f"preflight must not return an application record: {preflight}")
print(f"permission package preflight verified: canApply={expected_can_apply} check={expected_check}")
PY
}

assert_production_readiness() {
  local expected_status="$1"
  local expected_check="$2"
  RESPONSE_BODY="$HTTP_BODY" python3 - "$expected_status" "$expected_check" "$APPLICATION_ID" <<'PY'
import json
import os
import sys

readiness = json.loads(os.environ["RESPONSE_BODY"])["data"]
expected_status, expected_check, application_id = sys.argv[1:4]
if readiness.get("status") != expected_status:
    raise SystemExit(f"production readiness status={readiness.get('status')!r} want {expected_status!r}: {readiness}")
summary = readiness.get("summary") or {}
checks = {check.get("code"): check for check in readiness.get("checks", [])}
if expected_check not in checks:
    raise SystemExit(f"production readiness missing check {expected_check!r}: {checks}")
if expected_status == "ready":
    if summary.get("blockingCount") != 0:
        raise SystemExit(f"ready production readiness should have zero blockers: {summary}")
    expected_flags = {
        "hasApplication": True,
        "hasAllowedTrace": True,
        "hasDeniedTrace": True,
        "hasAppliedAudit": True,
        "accessProfileReady": True,
    }
    for key, value in expected_flags.items():
        if summary.get(key) is not value:
            raise SystemExit(f"production readiness summary[{key}]={summary.get(key)!r} want {value!r}: {summary}")
    application = readiness.get("latestApplication") or {}
    if application.get("id") != application_id:
        raise SystemExit(f"production readiness application id mismatch: {application}")
    runtime = readiness.get("runtimeEvidence") or {}
    if not runtime.get("allowedTrace") or not runtime.get("deniedTrace"):
        raise SystemExit(f"production readiness missing runtime records: {runtime}")
    audit = readiness.get("auditEvidence") or {}
    if not audit.get("appliedEvent"):
        raise SystemExit(f"production readiness missing applied audit record: {audit}")
else:
    if summary.get("blockingCount", 0) < 1:
        raise SystemExit(f"blocked production readiness should include blockers: {summary}")
    check = checks[expected_check]
    if check.get("severity") != "blocking":
        raise SystemExit(f"expected check {expected_check!r} to block, got {check}")
print(f"permission package production readiness verified: status={expected_status} check={expected_check}")
PY
}

assert_production_evidence_report() {
  local expected_status="$1"
  local expected_check="$2"
  RESPONSE_BODY="$HTTP_BODY" SYSTEM_INFO_BODY="$SYSTEM_INFO_BODY" python3 - "$expected_status" "$expected_check" "$APPLICATION_ID" "$CHILD_TENANT_ID" "$WORKSPACE_ID" <<'PY'
import json
import hashlib
import os
import re
import sys

report = json.loads(os.environ["RESPONSE_BODY"])["data"]
expected_status, expected_check, application_id, tenant_id, workspace_id = sys.argv[1:6]
if report.get("reportVersion") != "production-readiness-report/v1":
    raise SystemExit(f"unexpected report version: {report}")
if not isinstance(report.get("generatedBy"), str) or not report.get("generatedBy").strip():
    raise SystemExit(f"production report generatedBy should identify the exporting admin: {report.get('generatedBy')!r}")
report_digest = report.get("reportDigest")
if not re.fullmatch(r"[a-f0-9]{64}", str(report_digest or "")):
    raise SystemExit(f"production report digest should be a sha256 hex digest: {report_digest!r}")
if report.get("reportDigestAlgorithm") != "sha256-canonical-json-v1":
    raise SystemExit(f"production report digest algorithm={report.get('reportDigestAlgorithm')!r} want 'sha256-canonical-json-v1'")
digest_payload = dict(report)
digest_payload.pop("reportDigest", None)
expected_digest = hashlib.sha256(json.dumps(digest_payload, sort_keys=True, separators=(",", ":")).encode()).hexdigest()
if report_digest != expected_digest:
    raise SystemExit(f"production report digest={report_digest!r} want {expected_digest!r}")
if report.get("status") != expected_status:
    raise SystemExit(f"production report status={report.get('status')!r} want {expected_status!r}: {report}")
system_info = json.loads(os.environ["SYSTEM_INFO_BODY"]).get("data", {})
system_catalog = system_info.get("managementMcpToolCatalog") or {}
report_contract = report.get("platformContract") or {}
report_catalog = report_contract.get("managementMcpToolCatalog") or {}
if report_contract.get("apiVersion") != system_info.get("apiVersion"):
    raise SystemExit(f"production report apiVersion={report_contract.get('apiVersion')!r} does not match system info {system_info.get('apiVersion')!r}")
if report_catalog.get("metadataVersion") != system_catalog.get("metadataVersion"):
    raise SystemExit(f"production report catalog metadataVersion={report_catalog.get('metadataVersion')!r} does not match system info {system_catalog.get('metadataVersion')!r}")
if not re.fullmatch(r"[a-f0-9]{64}", str(report_catalog.get("catalogDigest", ""))):
    raise SystemExit(f"production report catalogDigest should be a sha256 hex digest: {report_catalog}")
if report_catalog.get("catalogDigest") != system_catalog.get("catalogDigest"):
    raise SystemExit(f"production report catalogDigest={report_catalog.get('catalogDigest')!r} does not match system info {system_catalog.get('catalogDigest')!r}")
scope = report.get("scope") or {}
if scope.get("tenantId") != tenant_id or scope.get("workspaceId") != workspace_id:
    raise SystemExit(f"unexpected production report scope: {scope}")
checks = {check.get("code"): check for check in report.get("checks", [])}
if expected_check not in checks:
    raise SystemExit(f"production report missing check {expected_check!r}: {checks}")
evidence = report.get("evidence") or {}
application = evidence.get("application") or {}
runtime = evidence.get("runtime") or {}
audit = evidence.get("audit") or {}
if expected_status == "ready":
    if application.get("id") != application_id or application.get("present") is not True:
        raise SystemExit(f"ready production report missing application record: {application}")
    if not runtime.get("allowedTraceId") or not runtime.get("deniedTraceId"):
        raise SystemExit(f"ready production report missing runtime records: {runtime}")
    if not audit.get("appliedEventId"):
        raise SystemExit(f"ready production report missing audit record: {audit}")
    if (evidence.get("accessProfile") or {}).get("present") is not True:
        raise SystemExit(f"ready production report missing access profile record: {evidence}")
else:
    if application.get("present") is True:
        raise SystemExit(f"blocked-before-apply report should not include an application record: {application}")
    if checks[expected_check].get("severity") != "blocking":
        raise SystemExit(f"expected report check {expected_check!r} to block, got {checks[expected_check]}")
print(f"permission package production report verified: status={expected_status} check={expected_check}")
PY
}

assert_listed_approval() {
  local expected_id="$1"
  local expected_status="${2:-pending}"
  RESPONSE_BODY="$HTTP_BODY" python3 - "$expected_id" "$expected_status" <<'PY'
import json
import os
import sys

rows = json.loads(os.environ["RESPONSE_BODY"])["data"]
expected_id, expected_status = sys.argv[1:3]
if not rows:
    raise SystemExit("expected one listed approval request")
if rows[0]["id"] != expected_id or rows[0]["status"] != expected_status:
    raise SystemExit(f"unexpected listed approval request: {rows[:1]}")
print("approval request list verified")
PY
}

assert_empty_approval_list() {
  RESPONSE_BODY="$HTTP_BODY" python3 <<'PY'
import json
import os

rows = json.loads(os.environ["RESPONSE_BODY"])["data"]
if rows:
    raise SystemExit(f"expected no approval requests, got {rows}")
print("approval request list is empty")
PY
}

assert_unconsumed_approval() {
  RESPONSE_BODY="$HTTP_BODY" python3 - "$APPROVAL_REQUEST_ID" <<'PY'
import json
import os
import sys

rows = json.loads(os.environ["RESPONSE_BODY"])["data"]
approval_request_id = sys.argv[1]
if not rows:
    raise SystemExit("expected one listed approved approval request")
approval = rows[0]
if approval["id"] != approval_request_id or approval["status"] != "approved":
    raise SystemExit(f"unexpected approved approval request: {approval}")
if approval.get("consumedByApplicationId") or approval.get("consumedAt"):
    raise SystemExit(f"preflight should not consume approval request: {approval}")
print("approval request remains unconsumed after preflight")
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
next_action_codes = doc["draft"]["policyGate"].get("nextActionCodes") or []
if "create_approval_request" not in next_action_codes:
    raise SystemExit(
        f"applied draft should preserve policy gate nextActionCodes={next_action_codes!r}: {doc['draft']['policyGate']}"
    )
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
print(f"{expected_decision} trace record verified")
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
    raise SystemExit(f"expected allowed and denied trace records, summary={summary}")
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

assert_application_health() {
  RESPONSE_BODY="$HTTP_BODY" python3 - "$APPLICATION_ID" <<'PY'
import json
import os
import sys

health = json.loads(os.environ["RESPONSE_BODY"])["data"]
application_id = sys.argv[1]
summary = health.get("summary") or {}
expected_summary = {
    "total": 1,
    "ready": 1,
    "drifted": 0,
    "needsReview": 0,
}
for key, value in expected_summary.items():
    if summary.get(key) != value:
        raise SystemExit(f"health summary[{key}]={summary.get(key)} want {value}; summary={summary}")
applications = health.get("applications") or []
if len(applications) != 1:
    raise SystemExit(f"expected exactly one application health row: {health}")
row = applications[0]
if row.get("application", {}).get("id") != application_id:
    raise SystemExit(f"health application id mismatch: {row}")
expected_row = {
    "status": "ready",
    "createdObjectCount": 6,
    "activeObjectCount": 6,
    "missingObjectCount": 0,
    "rollbackReady": True,
}
for key, value in expected_row.items():
    if row.get(key) != value:
        raise SystemExit(f"health row[{key}]={row.get(key)} want {value}; row={row}")
if row.get("blockerCodes") != []:
    raise SystemExit(f"health blockerCodes should be empty: {row}")
print("permission package application health verified")
PY
}

assert_application_impact() {
  RESPONSE_BODY="$HTTP_BODY" python3 - "$APPLICATION_ID" "$READ_TOOL" "$WRITE_TOOL" "$READ_CAPABILITY_ID" "$WRITE_CAPABILITY_ID" <<'PY'
import json
import os
import sys

impact = json.loads(os.environ["RESPONSE_BODY"])["data"]
application_id, read_tool, write_tool, read_capability_id, write_capability_id = sys.argv[1:6]
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
if review.get("blockerCodes") != []:
    raise SystemExit(f"rollback blocker codes should be an empty array: {review}")
if not review.get("steps"):
    raise SystemExit(f"rollback review should include steps: {review}")
plan = impact.get("remediationPlan") or {}
if plan.get("executionMode") != "read_only":
    raise SystemExit(f"remediation executionMode={plan.get('executionMode')!r}: {plan}")
if plan.get("ready") is not True:
    raise SystemExit(f"remediation plan should be ready: {plan}")
if plan.get("blockers") != []:
    raise SystemExit(f"remediation blockers should be an empty array: {plan}")
if plan.get("blockerCodes") != []:
    raise SystemExit(f"remediation blocker codes should be an empty array: {plan}")
actions = plan.get("actions") or []
if not actions:
    raise SystemExit(f"remediation plan should include actions: {plan}")
if any(action.get("readOnly") is not True for action in actions):
    raise SystemExit(f"all remediation actions must be read-only: {actions}")

def action_order(target_type, target_id, action_name):
    for row in actions:
        if row.get("targetType") == target_type and row.get("targetId") == target_id and row.get("action") == action_name:
            return int(row.get("order") or 0)
    return 0

for capability_id in (read_capability_id, write_capability_id):
    if not action_order("capability", capability_id, "manual_review"):
        raise SystemExit(f"missing capability manual review remediation action for {capability_id}: {actions}")
for object_type in ("tenant_entitlement", "workspace_assignment", "instance_assignment"):
    if not any(row.get("targetType") == object_type and row.get("action") == "disable" for row in actions):
        raise SystemExit(f"missing {object_type} disable remediation action: {actions}")
if not action_order("access_decision", application_id, "verify"):
    raise SystemExit(f"missing final access decision verification action: {actions}")
first_instance = min((int(row.get("order") or 0) for row in actions if row.get("targetType") == "instance_assignment" and row.get("action") == "disable"), default=0)
first_workspace = min((int(row.get("order") or 0) for row in actions if row.get("targetType") == "workspace_assignment" and row.get("action") == "disable"), default=0)
first_tenant = min((int(row.get("order") or 0) for row in actions if row.get("targetType") == "tenant_entitlement" and row.get("action") == "disable"), default=0)
if not (first_instance and first_workspace and first_tenant and first_instance < first_workspace < first_tenant):
    raise SystemExit(f"disable remediation actions should be ordered instance -> workspace -> tenant: {actions}")
print("permission package application impact verified")
PY
}

assert_application_impact_rehearsal() {
  RESPONSE_BODY="$HTTP_BODY" python3 - "$APPLICATION_ID" <<'PY'
import json
import os
import sys

impact = json.loads(os.environ["RESPONSE_BODY"])["data"]
application_id = sys.argv[1]
if impact["application"]["id"] != application_id:
    raise SystemExit(f"rehearsal application id mismatch: {impact['application']}")
rehearsal = impact.get("rehearsal") or {}
if rehearsal.get("enabled") is not True or rehearsal.get("scenario") != "grant_drift":
    raise SystemExit(f"expected grant_drift rehearsal metadata: {rehearsal}")
summary = impact["summary"]
if summary.get("createdObjectCount") != 6:
    raise SystemExit(f"rehearsal createdObjectCount={summary.get('createdObjectCount')}: {summary}")
if summary.get("activeObjectCount") >= summary.get("createdObjectCount"):
    raise SystemExit(f"rehearsal should reduce active object count: {summary}")
if summary.get("missingObjectCount") != 1:
    raise SystemExit(f"rehearsal should simulate one missing object: {summary}")
if summary.get("rollbackReady") is not False:
    raise SystemExit(f"rehearsal should not be rollback-ready: {summary}")
review = impact["rollbackReview"]
plan = impact.get("remediationPlan") or {}
for code in ("missing_created_objects", "inactive_created_objects"):
    if code not in (review.get("blockerCodes") or []):
        raise SystemExit(f"rehearsal rollback missing blocker code {code}: {review}")
    if code not in (plan.get("blockerCodes") or []):
        raise SystemExit(f"rehearsal remediation missing blocker code {code}: {plan}")
if review.get("ready") is not False or plan.get("ready") is not False:
    raise SystemExit(f"rehearsal drift should require review: rollback={review} remediation={plan}")
actions = plan.get("actions") or []
if not actions:
    raise SystemExit(f"rehearsal remediation plan should include actions: {plan}")
if any(action.get("readOnly") is not True for action in actions):
    raise SystemExit(f"all rehearsal actions must be read-only: {actions}")
for object_type in ("workspace_assignment", "instance_assignment"):
    if not any(row.get("targetType") == object_type and row.get("action") == "investigate" for row in actions):
        raise SystemExit(f"missing rehearsal investigate action for {object_type}: {actions}")
if not any(row.get("targetType") == "access_decision" and row.get("targetId") == application_id and row.get("action") == "verify" for row in actions):
    raise SystemExit(f"missing rehearsal final access verification action: {actions}")
print("permission package application drift rehearsal verified")
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
print("permission package approval audit record verified")
PY
}

start_mcp_server() {
  if [[ "$START_MOCK_MCP" != "true" ]]; then
    return
  fi
  if curl -fsS "http://${MOCK_MCP_HOST}:${MOCK_MCP_PORT}/healthz" >/dev/null 2>&1; then
    echo "MCP server already running on ${MOCK_MCP_HOST}:${MOCK_MCP_PORT}"
    return
  fi
  case "$MCP_SERVER_MODE" in
    mock)
      scripts/mock-mcp-server.py --host "$MOCK_MCP_HOST" --port "$MOCK_MCP_PORT" &
      MOCK_MCP_PID="$!"
      ;;
    real)
      need node
      need pnpm
      pnpm --dir scripts/real-mcp install --frozen-lockfile >/dev/null
      (cd scripts/real-mcp && REAL_MCP_HOST="$MOCK_MCP_HOST" REAL_MCP_PORT="$MOCK_MCP_PORT" node server.mjs) &
      MOCK_MCP_PID="$!"
      ;;
    *)
      echo "MCP_SERVER_MODE must be mock or real" >&2
      exit 1
      ;;
  esac
  for _ in $(seq 1 30); do
    if curl -fsS "http://${MOCK_MCP_HOST}:${MOCK_MCP_PORT}/healthz" >/dev/null 2>&1; then
      echo "started ${MCP_SERVER_MODE} MCP server on ${MCP_ENDPOINT}"
      return
    fi
    sleep 0.2
  done
  echo "MCP server did not become ready" >&2
  exit 1
}

need curl
need python3

echo "AgentHarbor permission package approval scenario"
echo "BASE_URL=$BASE_URL"
echo "MCP_ENDPOINT=$MCP_ENDPOINT"
echo "MCP_SERVER_MODE=$MCP_SERVER_MODE"
echo "RUN_ID=$RUN_ID"
echo "SUBJECT_ID=$SUBJECT_ID"
if [[ -n "$ADMIN_KEY" ]]; then
  echo "ADMIN_KEY=provided"
else
  echo "ADMIN_KEY=not set"
fi
if [[ -n "$REQUESTER_ADMIN_KEY" && -n "$REVIEWER_ADMIN_KEY" && "$REQUESTER_ADMIN_KEY" != "$REVIEWER_ADMIN_KEY" ]]; then
  echo "ADMIN_IDENTITIES=split requester/reviewer"
fi

start_mcp_server

request GET "/healthz"
expect_status 200 "AgentHarbor health check"

request GET "/api/v1/system/info"
expect_status 200 "system info compatibility summary"
SYSTEM_INFO_BODY="$HTTP_BODY"

request POST "/api/v1/management/mcp" "$(json_body tools-list)"
expect_status 200 "list management MCP tools with safety metadata"
assert_management_mcp_tool_safety

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

request POST "/api/v1/management/mcp" "$(mcp_explain_permission_package_body)"
expect_status 200 "explain approval-required permission package through management MCP"
assert_mcp_permission_package_explain

request POST "/api/v1/management/mcp" "$(mcp_explain_access_decision_body "$READ_CAPABILITY_ID")"
expect_status 200 "explain denied access through management MCP before permission package apply"
assert_mcp_access_explain "denied" "approve_capability"

request POST "/api/v1/permission-packages:preflight" "$(permission_package_body)"
expect_status 200 "preflight approval-required package without approval"
assert_permission_package_preflight "false" "approval_request_missing"

request POST "/api/v1/permission-packages:apply" "$(permission_package_body)"
expect_status 400 "reject approval-required package without approval"
echo "direct apply without approval rejected"

request POST "/api/v1/management/mcp" "$(mcp_permission_package_write_body create_permission_package_approval_request missing)"
expect_status 200 "reject management MCP approval request without write confirmation"
assert_mcp_tool_error "VALIDATION_FAILED" "confirmation.confirmed"

request GET "/api/v1/permission-packages/approval-requests?tenantId=$ROOT_TENANT_ID&workspaceId=$WORKSPACE_ID&templateId=$TEMPLATE_ID&targetId=$TARGET_ID&callerInstanceId=$CALLER_ID&status=pending&limit=1"
expect_status 200 "list approvals after rejected management MCP write"
assert_empty_approval_list

request POST "/api/v1/management/mcp" "$(mcp_permission_package_write_body create_permission_package_approval_request confirmed)"
expect_status 200 "create withdrawable approval request through management MCP"
assert_mcp_approval_request "pending"
WITHDRAWN_APPROVAL_REQUEST_ID="$(json_get result.structuredContent.id)"

request POST "/api/v1/management/mcp" "$(mcp_approval_resolution_write_body withdraw_permission_package_approval_request "$WITHDRAWN_APPROVAL_REQUEST_ID" "$REQUESTER_ACTOR" "wrong scope for this request")"
expect_status 200 "withdraw pending approval request through management MCP"
assert_mcp_approval_request "withdrawn" "$WITHDRAWN_APPROVAL_REQUEST_ID"

request GET "/api/v1/permission-packages/approval-requests?tenantId=$ROOT_TENANT_ID&workspaceId=$WORKSPACE_ID&templateId=$TEMPLATE_ID&targetId=$TARGET_ID&callerInstanceId=$CALLER_ID&status=withdrawn&limit=1"
expect_status 200 "list withdrawn approval request"
assert_listed_approval "$WITHDRAWN_APPROVAL_REQUEST_ID" "withdrawn"

request POST "/api/v1/permission-packages:apply" "$(permission_package_body "$WITHDRAWN_APPROVAL_REQUEST_ID")"
expect_status 400 "reject withdrawn approval request"
assert_body_contains "approved" "withdrawn approval apply"
echo "withdrawn approval request cannot authorize apply"

request POST "/api/v1/permission-packages/approval-requests" "$(permission_package_body)"
expect_status 201 "create approval request"
assert_approval_request "pending"
APPROVAL_REQUEST_ID="$(json_get data.id)"

request GET "/api/v1/permission-packages/approval-requests?tenantId=$ROOT_TENANT_ID&workspaceId=$WORKSPACE_ID&templateId=$TEMPLATE_ID&targetId=$TARGET_ID&callerInstanceId=$CALLER_ID&status=pending&limit=1"
expect_status 200 "list pending approval requests"
assert_listed_approval "$APPROVAL_REQUEST_ID"

if [[ -n "$REQUESTER_ADMIN_KEY" && -n "$REVIEWER_ADMIN_KEY" && "$REQUESTER_ADMIN_KEY" != "$REVIEWER_ADMIN_KEY" ]]; then
  request POST "/api/v1/permission-packages/approval-requests/$APPROVAL_REQUEST_ID/approve" "$(json_body approval-resolution "$APPROVAL_REVIEWER" "requester must not impersonate reviewer")" "" "" "" "$REQUESTER_ADMIN_KEY"
  expect_status 403 "reject approval reviewer impersonation"
  assert_body_contains "authenticated admin identity" "approval reviewer impersonation"
  echo "approval reviewer impersonation rejected"
fi

request POST "/api/v1/permission-packages/approval-requests/$APPROVAL_REQUEST_ID/approve" "$(json_body approval-resolution "$APPROVAL_REVIEWER" "approved for local production journey scenario")" "" "" "" "$REVIEWER_ADMIN_KEY"
expect_status 200 "approve approval request"
assert_approval_request "approved" "$APPROVAL_REQUEST_ID"

request POST "/api/v1/permission-packages:preflight" "$(permission_package_body "$APPROVAL_REQUEST_ID")"
expect_status 200 "preflight approved permission package"
assert_permission_package_preflight "true" "approval_request_ready"

request GET "/api/v1/permission-packages/approval-requests?tenantId=$ROOT_TENANT_ID&workspaceId=$WORKSPACE_ID&templateId=$TEMPLATE_ID&targetId=$TARGET_ID&callerInstanceId=$CALLER_ID&status=approved&limit=1"
expect_status 200 "list unconsumed approval request after preflight"
assert_unconsumed_approval

request GET "$(production_readiness_path "$APPROVAL_REQUEST_ID")"
expect_status 200 "check production readiness before apply"
assert_production_readiness "blocked" "application_present"

request GET "$(production_evidence_report_path "$APPROVAL_REQUEST_ID")"
expect_status 200 "export production report before apply"
assert_production_evidence_report "blocked" "application_present"

request POST "/api/v1/permission-packages:apply" "$(permission_package_body "$APPROVAL_REQUEST_ID")"
expect_status 201 "apply approved permission package"
APPLICATION_ID="$(assert_apply_response "$APPROVAL_REQUEST_ID")"
echo "applied permission package application: $APPLICATION_ID"

request GET "/api/v1/permission-packages/approval-requests?tenantId=$ROOT_TENANT_ID&workspaceId=$WORKSPACE_ID&templateId=$TEMPLATE_ID&targetId=$TARGET_ID&callerInstanceId=$CALLER_ID&status=approved&limit=1"
expect_status 200 "list consumed approval request"
assert_consumed_approval

request POST "/api/v1/permission-packages:preflight" "$(permission_package_body)"
expect_status 200 "preflight already-applied permission package"
assert_permission_package_preflight "false" "application_already_applied"

request POST "/api/v1/management/mcp" "$(mcp_explain_access_decision_body "$READ_CAPABILITY_ID")"
expect_status 200 "explain allowed access through management MCP after permission package apply"
assert_mcp_access_explain "allowed" "no_change_required"

request POST "/api/v1/permission-packages:apply" "$(permission_package_body "$APPROVAL_REQUEST_ID")"
expect_status 400 "reject consumed approval request"
assert_body_contains "PERMISSION_PACKAGE_APPROVAL_ALREADY_CONSUMED" "consumed approval retry error code"
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

request GET "/api/v1/permission-packages/applications/health?tenantId=$ROOT_TENANT_ID&workspaceId=$WORKSPACE_ID&templateId=$TEMPLATE_ID&targetId=$TARGET_ID&callerInstanceId=$CALLER_ID&limit=1"
expect_status 200 "inspect permission package application health"
assert_application_health

request GET "/api/v1/permission-packages/applications/$APPLICATION_ID/impact?tenantId=$ROOT_TENANT_ID&workspaceId=$WORKSPACE_ID"
expect_status 200 "review permission package application impact"
assert_application_impact

request GET "/api/v1/permission-packages/applications/$APPLICATION_ID/impact?tenantId=$ROOT_TENANT_ID&workspaceId=$WORKSPACE_ID&rehearsal=grant_drift"
expect_status 200 "rehearse permission package grant drift"
assert_application_impact_rehearsal

request GET "/api/v1/permission-packages/applications/$APPLICATION_ID/impact?tenantId=$ROOT_TENANT_ID&workspaceId=$WORKSPACE_ID"
expect_status 200 "review permission package application impact after rehearsal"
assert_application_impact

request GET "/api/v1/audit/events?action=permission_package.applied&resourceId=$APPLICATION_ID&limit=1"
expect_status 200 "list applied audit events"
assert_applied_audit_event

request GET "$(production_readiness_path)"
expect_status 200 "check production readiness after runtime records"
assert_production_readiness "ready" "runtime_allowed_trace_present"

request GET "$(production_evidence_report_path)"
expect_status 200 "export production report after runtime records"
assert_production_evidence_report "ready" "runtime_allowed_trace_present"

echo "permission package approval scenario complete"
