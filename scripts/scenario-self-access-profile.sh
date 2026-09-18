#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:9090}"
ADMIN_KEY="${ADMIN_KEY:-}"
RUN_ID="${RUN_ID:-self-access-profile-$(date +%Y%m%d%H%M%S)}"
TENANT_ID="${TENANT_ID:-tenant-self-${RUN_ID}}"
WORKSPACE_ID="${WORKSPACE_ID:-ws-self-access-profile}"
MCP_ENDPOINT="${MCP_ENDPOINT:-}"
ALLOWED_TOOL="${ALLOWED_TOOL:-search_customer}"
SUBJECT_ID="${SUBJECT_ID:-user:self-access-profile}"

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
    tenant_id, name = sys.argv[2], sys.argv[3]
    body = {"id": tenant_id, "name": name, "status": "active"}
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
    workspace_assignment_id, caller_id, subject_selector = sys.argv[2], sys.argv[3], sys.argv[4]
    body = {
        "workspaceAssignmentId": workspace_assignment_id,
        "callerInstanceId": caller_id,
        "subjectSelector": subject_selector,
        "effect": "allow",
        "status": "enabled",
        "dataScopes": [{"field": "email"}],
    }
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

expect_error_reason() {
  local reason="$1"
  local label="$2"
  RESPONSE_BODY="$HTTP_BODY" python3 - "$reason" "$label" <<'PY'
import json
import os
import sys

reason, label = sys.argv[1], sys.argv[2]
doc = json.loads(os.environ["RESPONSE_BODY"])
message = doc.get("message", "") or doc.get("error", {}).get("message", "")
if reason not in message:
    raise SystemExit(f"{label}: expected message containing {reason!r}, got {message!r}")
PY
}

assert_empty_self_profile() {
  RESPONSE_BODY="$HTTP_BODY" python3 - "$TENANT_ID" "$SUBJECT_ID" <<'PY'
import json
import os
import sys

tenant_id, subject_id = sys.argv[1], sys.argv[2]
profile = json.loads(os.environ["RESPONSE_BODY"])["data"]
if profile["caller"]["tenantId"] != tenant_id:
    raise SystemExit(f"caller tenant mismatch: {profile['caller']}")
if profile["subjectId"] != subject_id:
    raise SystemExit(f"subject mismatch: {profile['subjectId']}")
if profile["key"]["kind"] != "agent":
    raise SystemExit(f"expected agent key kind, got {profile['key']}")
if profile["targets"] != []:
    raise SystemExit(f"expected no targets without grants, got {profile['targets']}")
print("empty self access profile verified")
PY
}

assert_self_profile_boundary() {
  RESPONSE_BODY="$HTTP_BODY" python3 - "$TENANT_ID" "$TARGET_ID" "$ALLOWED_CAPABILITY_ID" "$ALLOWED_TOOL" <<'PY'
import json
import os
import sys

tenant_id, target_id, capability_id, tool_key = sys.argv[1:5]
profile = json.loads(os.environ["RESPONSE_BODY"])["data"]
if profile["caller"]["tenantId"] != tenant_id:
    raise SystemExit(f"caller tenant mismatch: {profile['caller']}")
if profile["key"]["kind"] != "agent" or profile.get("handoff") is not None:
    raise SystemExit(f"expected ordinary key profile, got {profile['key']} handoff={profile.get('handoff')}")
if len(profile["targets"]) != 1 or profile["targets"][0]["targetId"] != target_id:
    raise SystemExit(f"expected exactly the granted target, got {profile['targets']}")
capabilities = profile["targets"][0]["capabilities"]
if len(capabilities) != 1 or capabilities[0]["id"] != capability_id or capabilities[0]["key"] != tool_key:
    raise SystemExit(f"expected exactly the granted capability, got {capabilities}")
if not profile["key"]["expiresAt"]:
    raise SystemExit(f"expected key expiry to be visible, got {profile['key']}")
print("self access profile boundary verified")
PY
}

need curl
need python3

echo "AgentHarbor self access profile scenario"
echo "BASE_URL=$BASE_URL"
echo "RUN_ID=$RUN_ID"

request GET "/healthz"
expect_status 200 "health check"

request GET "/api/v1/self/access-profile"
expect_status 401 "reject unauthenticated self profile"
expect_error_reason "missing bearer token" "unauthenticated self profile"

request POST "/api/v1/tenants" "$(json_body tenant "$TENANT_ID" "Self Access Profile Tenant")"
expect_status 201 "create tenant"

request POST "/api/v1/agents" "$(json_body agent "$TENANT_ID" "$WORKSPACE_ID" "Self Access Profile Caller" "local")"
expect_status 201 "create caller"
CALLER_ID="$(json_get data.id)"

request POST "/api/v1/agent-keys" "{\"agentId\":\"$CALLER_ID\",\"name\":\"self-access-profile key\"}"
expect_status 201 "create key"
AGENT_KEY_ID="$(json_get data.id)"
AGENT_KEY="$(json_get data.key)"

request GET "/api/v1/self/access-profile" "" "$AGENT_KEY" "" "$SUBJECT_ID"
expect_status 200 "fetch empty self access profile"
assert_empty_self_profile

if [[ -z "$MCP_ENDPOINT" ]]; then
  echo "MCP_ENDPOINT is not set; empty profile and auth diagnostics path verified. Set MCP_ENDPOINT to verify the capability boundary."
  exit 0
fi

request POST "/api/v1/agents" "$(json_body agent "$TENANT_ID" "$WORKSPACE_ID" "Self Access Profile MCP" "mcp" "$MCP_ENDPOINT")"
expect_status 201 "create mcp target"
TARGET_ID="$(json_get data.id)"

request POST "/api/v1/targets/$TARGET_ID/capabilities:refresh"
expect_status 200 "refresh capabilities"
ALLOWED_CAPABILITY_ID="$(capability_id_for_key "$ALLOWED_TOOL")"

request PATCH "/api/v1/capabilities/$ALLOWED_CAPABILITY_ID" "$(json_body approve-scoped "$TENANT_ID")"
expect_status 200 "approve scoped capability"

request POST "/api/v1/tenant-entitlements" "$(json_body entitlement "$TENANT_ID" "$TARGET_ID" "$ALLOWED_CAPABILITY_ID")"
expect_status 201 "create tenant entitlement"
ENTITLEMENT_ID="$(json_get data.id)"

request POST "/api/v1/workspace-assignments" "$(json_body workspace-assignment "$ENTITLEMENT_ID" "$WORKSPACE_ID")"
expect_status 201 "create workspace assignment"
WORKSPACE_ASSIGNMENT_ID="$(json_get data.id)"

request POST "/api/v1/instance-assignments" "$(json_body instance-assignment "$WORKSPACE_ASSIGNMENT_ID" "$CALLER_ID" "user:*")"
expect_status 201 "create instance assignment"

request GET "/api/v1/self/access-profile" "" "$AGENT_KEY" "" "$SUBJECT_ID"
expect_status 200 "fetch self access profile"
assert_self_profile_boundary

request GET "/api/v1/self/access-profile" "" "ah-unknown-token" "" "$SUBJECT_ID"
expect_status 401 "reject unknown token"
expect_error_reason "invalid or expired bearer token" "unknown token"

request POST "/api/v1/agent-keys" "{\"agentId\":\"$CALLER_ID\",\"name\":\"self-access-profile short-lived key\",\"expiresInSeconds\":1}"
expect_status 201 "create short-lived key"
SHORT_LIVED_KEY="$(json_get data.key)"
sleep 2
request GET "/api/v1/self/access-profile" "" "$SHORT_LIVED_KEY" "" "$SUBJECT_ID"
expect_status 401 "reject expired token"
expect_error_reason "bearer token has expired" "expired token"

request DELETE "/api/v1/api-keys/$AGENT_KEY_ID"
expect_status 200 "revoke agent key"
request GET "/api/v1/self/access-profile" "" "$AGENT_KEY" "" "$SUBJECT_ID"
expect_status 401 "reject revoked token"
expect_error_reason "bearer token has been revoked" "revoked token"

echo "self access profile scenario complete"
