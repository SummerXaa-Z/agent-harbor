#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT_DIR}/scripts/lib/ports.sh"

BASE_URL="${BASE_URL:-http://127.0.0.1:9196}"
API_ADDR="${BASE_URL#http://}"
API_PORT="${API_ADDR##*:}"
MOCK_MCP_HOST="${MOCK_MCP_HOST:-127.0.0.1}"
MOCK_MCP_PORT="${MOCK_MCP_PORT:-8796}"
MCP_ENDPOINT="http://${MOCK_MCP_HOST}:${MOCK_MCP_PORT}/mcp"
RUN_ID="tenant-permission-center-$(date +%Y%m%d%H%M%S)"
PLATFORM_KEY="platform-key-${RUN_ID}"
TENANT_ROOT="tenant-root-${RUN_ID}"
TENANT_CHILD="tenant-child-${RUN_ID}"
WORKSPACE_ID="ws-support-${RUN_ID}"
PIDS=()

cleanup() {
  for pid in "${PIDS[@]:-}"; do
    kill "${pid}" >/dev/null 2>&1 || true
  done
  wait >/dev/null 2>&1 || true
}
trap cleanup EXIT

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing dependency: $1" >&2
    exit 1
  fi
}

wait_url() {
  local url="$1"
  local label="$2"
  for _ in $(seq 1 80); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return
    fi
    sleep 0.1
  done
  echo "${label} did not become ready" >&2
  exit 1
}

need curl
need go
need jq
need python3

echo "AgentHarbor tenant permission center scenario"
echo "BASE_URL=${BASE_URL}"
echo "MCP_ENDPOINT=${MCP_ENDPOINT}"
echo "RUN_ID=${RUN_ID}"

if [[ "${API_PORT}" =~ ^[0-9]+$ ]]; then
  assert_port_free "API" "${API_PORT}"
fi
assert_port_free "MCP" "${MOCK_MCP_PORT}"

python3 "${ROOT_DIR}/scripts/mock-mcp-server.py" --host "${MOCK_MCP_HOST}" --port "${MOCK_MCP_PORT}" >/tmp/agent-harbor-tenant-center-mcp-${RUN_ID}.log 2>&1 &
PIDS+=("$!")
wait_url "http://${MOCK_MCP_HOST}:${MOCK_MCP_PORT}/healthz" "mock MCP server"

cd "${ROOT_DIR}"
AGENT_HARBOR_ADDR="${API_ADDR}" \
AGENT_HARBOR_ADMIN_IDENTITIES="platform=${PLATFORM_KEY}|role=platform_admin" \
AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true \
AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true \
AGENT_HARBOR_SESSION_SECRET="scenario-session-${RUN_ID}" \
AGENT_HARBOR_DATABASE_URL= \
AGENT_HARBOR_CREDENTIAL_KEY= \
  go run ./cmd/agent-harbor >"/tmp/agent-harbor-tenant-center-${RUN_ID}.log" 2>&1 &
PIDS+=("$!")
wait_url "${BASE_URL}/healthz" "AgentHarbor API"

curl_json() {
  local method="$1"
  local path="$2"
  local key="$3"
  local body="${4:-}"
  if [[ -n "${body}" ]]; then
    curl -fsS -X "${method}" "${BASE_URL}${path}" -H "Content-Type: application/json" -H "X-Admin-Key: ${key}" --data "${body}"
  else
    curl -fsS -X "${method}" "${BASE_URL}${path}" -H "X-Admin-Key: ${key}"
  fi
}

curl_json POST /api/v1/tenants "${PLATFORM_KEY}" "{\"id\":\"${TENANT_ROOT}\",\"name\":\"Platform Operations\"}" >/dev/null
curl_json POST /api/v1/tenants "${PLATFORM_KEY}" "{\"id\":\"${TENANT_CHILD}\",\"parentTenantId\":\"${TENANT_ROOT}\",\"name\":\"Customer Service\"}" >/dev/null
CALLER=$(curl_json POST /api/v1/agents "${PLATFORM_KEY}" "{\"tenantId\":\"${TENANT_CHILD}\",\"workspaceId\":\"${WORKSPACE_ID}\",\"name\":\"Support Assistant\",\"channelType\":\"local\",\"status\":\"active\"}" | jq -r '.data.id')
TARGET=$(curl_json POST /api/v1/agents "${PLATFORM_KEY}" "{\"tenantId\":\"${TENANT_CHILD}\",\"workspaceId\":\"${WORKSPACE_ID}\",\"name\":\"Ticket Tool Service\",\"channelType\":\"mcp\",\"channelConfig\":{\"endpoint\":\"${MCP_ENDPOINT}\"},\"status\":\"active\"}" | jq -r '.data.id')
CAP=$(curl_json POST "/api/v1/targets/${TARGET}/capabilities:refresh" "${PLATFORM_KEY}" '{}' | jq -r '.data[0].id')
ENTITLEMENT=$(curl_json POST /api/v1/tenant-entitlements "${PLATFORM_KEY}" "{\"tenantId\":\"${TENANT_CHILD}\",\"targetId\":\"${TARGET}\",\"capabilityId\":\"${CAP}\",\"effect\":\"allow\",\"status\":\"enabled\",\"dataScopes\":[{\"dataDomain\":\"support\",\"dataset\":\"tickets\",\"region\":\"us-east\"}]}" | jq -r '.data.id')
WORKSPACE_ASSIGNMENT=$(curl_json POST /api/v1/workspace-assignments "${PLATFORM_KEY}" "{\"tenantEntitlementId\":\"${ENTITLEMENT}\",\"workspaceId\":\"${WORKSPACE_ID}\",\"effect\":\"allow\",\"status\":\"enabled\",\"dataScopes\":[{\"dataDomain\":\"support\",\"dataset\":\"tickets\",\"region\":\"us-east\"}]}" | jq -r '.data.id')
curl_json POST /api/v1/instance-assignments "${PLATFORM_KEY}" "{\"workspaceAssignmentId\":\"${WORKSPACE_ASSIGNMENT}\",\"callerInstanceId\":\"${CALLER}\",\"subjectSelector\":\"user:support-*\",\"effect\":\"allow\",\"status\":\"enabled\",\"dataScopes\":[{\"dataDomain\":\"support\",\"dataset\":\"tickets\",\"region\":\"us-east\"}]}" >/dev/null
ADMIN_CREATE=$(curl_json POST /api/v1/admin-identities "${PLATFORM_KEY}" "{\"actor\":\"tenant-admin-${RUN_ID}\",\"displayName\":\"Tenant Admin\",\"role\":\"tenant_admin\",\"tenantId\":\"${TENANT_CHILD}\",\"workspaceId\":\"${WORKSPACE_ID}\"}")
TENANT_ADMIN_KEY=$(printf '%s' "${ADMIN_CREATE}" | jq -r '.data.key')

CENTER=$(curl_json GET "/api/v1/tenants/${TENANT_CHILD}/permission-center" "${PLATFORM_KEY}")
printf '%s' "${CENTER}" | jq -e '.data.operatorBoundary.canManageAdministrators == true' >/dev/null
printf '%s' "${CENTER}" | jq -e '.data.administrators | length >= 1' >/dev/null
printf '%s' "${CENTER}" | jq -e '.data.workspaces[0].workspaceId == "'"${WORKSPACE_ID}"'"' >/dev/null
printf '%s' "${CENTER}" | jq -e '.data.capabilities | length >= 1' >/dev/null
printf '%s' "${CENTER}" | jq -e '.data.capabilities[0].dataScopes[0].dataDomain == "support"' >/dev/null

SCOPED_CENTER=$(curl_json GET "/api/v1/tenants/${TENANT_CHILD}/permission-center" "${TENANT_ADMIN_KEY}")
printf '%s' "${SCOPED_CENTER}" | jq -e '.data.operatorBoundary.canManageAdministrators == false' >/dev/null
printf '%s' "${SCOPED_CENTER}" | jq -e '.data.operatorBoundary.tenantId == "'"${TENANT_CHILD}"'"' >/dev/null
printf '%s' "${SCOPED_CENTER}" | jq -e '.data.operatorBoundary.workspaceId == "'"${WORKSPACE_ID}"'"' >/dev/null

if curl -fsS -X GET "${BASE_URL}/api/v1/tenants/${TENANT_ROOT}/permission-center" -H "X-Admin-Key: ${TENANT_ADMIN_KEY}" >/tmp/tenant-center-widen.json 2>/dev/null; then
  echo "tenant admin unexpectedly fetched parent tenant center" >&2
  exit 1
fi

echo "tenant permission center scenario complete"
