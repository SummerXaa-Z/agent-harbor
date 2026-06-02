#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:9090}"
ADMIN_KEY="${ADMIN_KEY:-}"
RUN_ID="${RUN_ID:-sprint14-$(date +%Y%m%d%H%M%S)}"
ROOT_TENANT_ID="${ROOT_TENANT_ID:-tenant-root-${RUN_ID}}"
CHILD_TENANT_ID="${CHILD_TENANT_ID:-tenant-child-${RUN_ID}}"
GRANDCHILD_TENANT_ID="${GRANDCHILD_TENANT_ID:-tenant-grandchild-${RUN_ID}}"
UNRELATED_TENANT_ID="${UNRELATED_TENANT_ID:-tenant-unrelated-${RUN_ID}}"
WORKSPACE_ID="${WORKSPACE_ID:-ws-sprint14}"
MCP_ENDPOINT="${MCP_ENDPOINT:-}"
ALLOWED_TOOL="${ALLOWED_TOOL:-search_customer}"

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
elif kind == "approve":
    body = {"discoveryStatus": "approved"}
elif kind == "entitlement":
    tenant_id, target_id, capability_id = sys.argv[2], sys.argv[3], sys.argv[4]
    body = {
        "tenantId": tenant_id,
        "targetId": target_id,
        "capabilityId": capability_id,
        "effect": "allow",
        "status": "enabled",
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

assert_tenant_count() {
  local expected="$1"
  RESPONSE_BODY="$HTTP_BODY" python3 - "$expected" <<'PY'
import json
import os
import sys

expected = int(sys.argv[1])
doc = json.loads(os.environ["RESPONSE_BODY"])
rows = doc["data"]
if len(rows) != expected:
    raise SystemExit(f"expected {expected} rows, got {len(rows)}: {rows}")
print(f"verified {expected} scoped rows")
PY
}

need curl
need python3

echo "AgentHarbor Sprint 14 tenant hierarchy demo"
echo "BASE_URL=$BASE_URL"
echo "RUN_ID=$RUN_ID"

request GET "/healthz"
expect_status 200 "health check"

request POST "/api/v1/tenants" "$(json_body tenant "$ROOT_TENANT_ID" "" "Sprint14 Root")"
expect_status 201 "create root tenant"
request POST "/api/v1/tenants" "$(json_body tenant "$CHILD_TENANT_ID" "$ROOT_TENANT_ID" "Sprint14 Child")"
expect_status 201 "create child tenant"
request POST "/api/v1/tenants" "$(json_body tenant "$GRANDCHILD_TENANT_ID" "$CHILD_TENANT_ID" "Sprint14 Grandchild")"
expect_status 201 "create grandchild tenant"
request POST "/api/v1/tenants" "$(json_body tenant "$UNRELATED_TENANT_ID" "" "Sprint14 Unrelated")"
expect_status 201 "create unrelated tenant"

request POST "/api/v1/tenants" "$(json_body tenant "tenant-level4-${RUN_ID}" "$GRANDCHILD_TENANT_ID" "Too Deep")"
expect_status 400 "reject fourth-level tenant"
echo "fourth-level tenant rejected"

request POST "/api/v1/agents" "$(json_body agent "$ROOT_TENANT_ID" "$WORKSPACE_ID" "Sprint14 Root Agent" "local")"
expect_status 201 "create root agent"
request POST "/api/v1/agents" "$(json_body agent "$CHILD_TENANT_ID" "$WORKSPACE_ID" "Sprint14 Child Agent" "local")"
expect_status 201 "create child agent"
request POST "/api/v1/agents" "$(json_body agent "$GRANDCHILD_TENANT_ID" "$WORKSPACE_ID" "Sprint14 Grandchild Agent" "local")"
expect_status 201 "create grandchild agent"

request GET "/api/v1/tenants?tenantId=$ROOT_TENANT_ID"
expect_status 200 "list tenant subtree"
assert_tenant_count 3

request GET "/api/v1/agents?tenantId=$ROOT_TENANT_ID&workspaceId=$WORKSPACE_ID"
expect_status 200 "list scoped agents"
assert_tenant_count 3

if [[ -z "$MCP_ENDPOINT" ]]; then
  echo "MCP_ENDPOINT is not set; tenant hierarchy demo complete. Set MCP_ENDPOINT to verify parent-to-child capability entitlement."
  exit 0
fi

request POST "/api/v1/agents" "$(json_body agent "$ROOT_TENANT_ID" "$WORKSPACE_ID" "Sprint14 Root MCP" "mcp" "$MCP_ENDPOINT")"
expect_status 201 "create root mcp target"
TARGET_ID="$(json_get data.id)"

request POST "/api/v1/targets/$TARGET_ID/capabilities:refresh"
expect_status 200 "refresh capabilities"
CAPABILITY_ID="$(capability_id_for_key "$ALLOWED_TOOL")"

request PATCH "/api/v1/capabilities/$CAPABILITY_ID" "$(json_body approve)"
expect_status 200 "approve capability"

request POST "/api/v1/tenant-entitlements" "$(json_body entitlement "$CHILD_TENANT_ID" "$TARGET_ID" "$CAPABILITY_ID")"
expect_status 201 "grant root target capability to child tenant"
echo "parent-to-child capability entitlement verified"

request POST "/api/v1/tenant-entitlements" "$(json_body entitlement "$UNRELATED_TENANT_ID" "$TARGET_ID" "$CAPABILITY_ID")"
expect_status 400 "reject unrelated tenant entitlement"
echo "unrelated tenant entitlement rejected"

echo "tenant hierarchy demo complete"
