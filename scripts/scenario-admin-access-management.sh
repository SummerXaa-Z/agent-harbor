#!/usr/bin/env bash
set -euo pipefail

API_HOST="${AGENT_HARBOR_ADMIN_ACCESS_API_HOST:-127.0.0.1}"
API_PORT="${AGENT_HARBOR_ADMIN_ACCESS_API_PORT:-9195}"
API_ADDR="${AGENT_HARBOR_ADMIN_ACCESS_API_ADDR:-${API_HOST}:${API_PORT}}"
BASE_URL="${AGENT_HARBOR_ADMIN_ACCESS_BASE_URL:-http://${API_HOST}:${API_PORT}}"
RUN_ID="${RUN_ID:-admin-access-$(date +%Y%m%d%H%M%S)}"
PLATFORM_KEY="${PLATFORM_KEY:-platform-key-${RUN_ID}}"
ROOT_TENANT_ID="tenant-root-${RUN_ID}"
EAST_TENANT_ID="tenant-east-${RUN_ID}"
WEST_TENANT_ID="tenant-west-${RUN_ID}"
EAST_WORKSPACE_ID="ws-support-${RUN_ID}"
WEST_WORKSPACE_ID="ws-finance-${RUN_ID}"
MANAGED_ACTOR="managed-east-admin-${RUN_ID}"
MANAGED_ID=""
MANAGED_KEY=""
MANAGED_SESSION_COOKIE=""
MANAGED_SESSION_CSRF=""
ROTATED_KEY=""

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
LOG_DIR="${TMPDIR:-/tmp}/agent-harbor-admin-access-${RUN_ID}"
PIDS=()
HTTP_STATUS=""
HTTP_BODY=""

cleanup() {
  local pid
  for pid in "${PIDS[@]:-}"; do
    kill "$pid" >/dev/null 2>&1 || true
  done
  wait >/dev/null 2>&1 || true
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing dependency: $1" >&2
    exit 1
  fi
}

port_in_use() {
  local port="$1"
  if command -v lsof >/dev/null 2>&1; then
    lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1
    return
  fi
  python3 - "$port" <<'PY'
import socket
import sys

port = int(sys.argv[1])
sock = socket.socket()
try:
    sock.bind(("127.0.0.1", port))
except OSError:
    raise SystemExit(0)
finally:
    sock.close()
raise SystemExit(1)
PY
}

show_logs() {
  local file
  for file in "$LOG_DIR"/*.log; do
    [[ -f "$file" ]] || continue
    echo "== $file ==" >&2
    tail -80 "$file" >&2 || true
  done
}

wait_http() {
  local label="$1"
  local url="$2"
  local i
  for i in $(seq 1 60); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      echo "$label ready: $url"
      return
    fi
    sleep 0.5
  done
  echo "$label did not become ready: $url" >&2
  show_logs
  exit 1
}

request() {
  local method="$1"
  local path="$2"
  local auth_mode="${3:-platform}"
  local body="${4:-}"
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

  case "$auth_mode" in
    platform)
      args+=(-H "X-Admin-Key: $PLATFORM_KEY")
      ;;
    managed)
      args+=(-H "X-Admin-Key: $MANAGED_KEY")
      ;;
    rotated)
      args+=(-H "X-Admin-Key: $ROTATED_KEY")
      ;;
    none)
      ;;
    *)
      rm -f "$tmp"
      echo "unknown auth mode: $auth_mode" >&2
      exit 1
      ;;
  esac

  if [[ -n "$body" ]]; then
    args+=(-d "$body")
  fi

  if ! HTTP_STATUS="$(curl "${args[@]}")"; then
    rm -f "$tmp"
    echo "curl failed for $method $path" >&2
    show_logs
    exit 1
  fi
  HTTP_BODY="$(<"$tmp")"
  rm -f "$tmp"
}

request_managed_login_session() {
  local key="$1"
  local tmp
  local headers
  tmp="$(mktemp)"
  headers="$(mktemp)"

  if ! HTTP_STATUS="$(curl -sS -D "$headers" -o "$tmp" -w "%{http_code}" \
    -X POST "$BASE_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "$(json_body login "$key")")"; then
    rm -f "$tmp" "$headers"
    echo "curl failed for managed console login" >&2
    show_logs
    exit 1
  fi
  HTTP_BODY="$(<"$tmp")"
  MANAGED_SESSION_COOKIE="$(tr -d '\r' < "$headers" | sed -n 's/^Set-Cookie: \(agent_harbor_session=[^;]*\).*/\1/p' | head -n 1)"
  MANAGED_SESSION_CSRF="$(json_get data.csrfToken)"
  rm -f "$tmp" "$headers"
  if [[ -z "$MANAGED_SESSION_COOKIE" || -z "$MANAGED_SESSION_CSRF" ]]; then
    echo "managed console login did not return session cookie and CSRF token" >&2
    echo "$HTTP_BODY" >&2
    show_logs
    exit 1
  fi
}

request_with_managed_session() {
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
    -H "Cookie: $MANAGED_SESSION_COOKIE"
    -H "X-AgentHarbor-CSRF: $MANAGED_SESSION_CSRF"
  )

  if [[ -n "$body" ]]; then
    args+=(-d "$body")
  fi

  if ! HTTP_STATUS="$(curl "${args[@]}")"; then
    rm -f "$tmp"
    echo "curl failed for managed session $method $path" >&2
    show_logs
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
    show_logs
    exit 1
  fi
}

assert_body_contains() {
  local expected="$1"
  local label="$2"
  if [[ "$HTTP_BODY" != *"$expected"* ]]; then
    echo "expected $label body to contain $expected" >&2
    echo "$HTTP_BODY" >&2
    show_logs
    exit 1
  fi
}

json_body() {
  python3 - "$@" <<'PY'
import json
import sys

kind = sys.argv[1]
if kind == "admin":
    actor, display_name, tenant_id, workspace_id = sys.argv[2], sys.argv[3], sys.argv[4], sys.argv[5]
    body = {
        "actor": actor,
        "displayName": display_name,
        "role": "tenant_admin",
        "tenantId": tenant_id,
        "workspaceId": workspace_id,
    }
elif kind == "login":
    body = {"adminKey": sys.argv[2]}
elif kind == "tenant":
    tenant_id, parent_id, name = sys.argv[2], sys.argv[3], sys.argv[4]
    body = {"id": tenant_id, "name": name, "status": "active"}
    if parent_id:
        body["parentTenantId"] = parent_id
elif kind == "agent":
    tenant_id, workspace_id, name = sys.argv[2], sys.argv[3], sys.argv[4]
    body = {
        "tenantId": tenant_id,
        "workspaceId": workspace_id,
        "name": name,
        "channelType": "local",
        "status": "active",
    }
else:
    raise SystemExit(f"unknown body kind: {kind}")
print(json.dumps(body, separators=(",", ":")))
PY
}

json_get() {
  RESPONSE_BODY="$HTTP_BODY" python3 - "$1" <<'PY'
import json
import os
import sys

value = json.loads(os.environ["RESPONSE_BODY"])
for part in sys.argv[1].split("."):
    value = value[part]
if value is None:
    print("")
else:
    print(value)
PY
}

assert_login_session() {
  local actor="$1"
  local role="$2"
  local tenant_id="$3"
  local workspace_id="$4"
  RESPONSE_BODY="$HTTP_BODY" python3 - "$actor" "$role" "$tenant_id" "$workspace_id" <<'PY'
import json
import os
import sys

expected_actor, expected_role, expected_tenant, expected_workspace = sys.argv[1:5]
session = json.loads(os.environ["RESPONSE_BODY"])["data"]
checks = {
    "actor": expected_actor,
    "role": expected_role,
    "tenantId": expected_tenant,
    "workspaceId": expected_workspace,
}
for key, expected in checks.items():
    if session.get(key) != expected:
        raise SystemExit(f"expected session {key}={expected!r}, got {session!r}")
print(f"verified managed admin login session for {expected_actor}")
PY
}

assert_agents_scope() {
  local expected_count="$1"
  local expected_tenant="$2"
  local expected_workspace="$3"
  RESPONSE_BODY="$HTTP_BODY" python3 - "$expected_count" "$expected_tenant" "$expected_workspace" <<'PY'
import json
import os
import sys

expected_count = int(sys.argv[1])
expected_tenant = sys.argv[2]
expected_workspace = sys.argv[3]
rows = json.loads(os.environ["RESPONSE_BODY"])["data"]
if len(rows) != expected_count:
    raise SystemExit(f"expected {expected_count} agent rows, got {len(rows)}: {rows}")
for row in rows:
    if row.get("tenantId") != expected_tenant or row.get("workspaceId") != expected_workspace:
        raise SystemExit(f"agent escaped expected scope {expected_tenant}/{expected_workspace}: {row}")
print(f"verified {expected_count} agent rows stay in {expected_tenant}/{expected_workspace}")
PY
}

assert_body_excludes_secret() {
  local label="$1"
  local secret="$2"
  RESPONSE_BODY="$HTTP_BODY" SECRET="$secret" python3 - "$label" <<'PY'
import os
import sys

label = sys.argv[1]
body = os.environ["RESPONSE_BODY"]
secret = os.environ["SECRET"]
if secret and secret in body:
    raise SystemExit(f"{label} leaked generated plaintext key")
if "keyHash" in body:
    raise SystemExit(f"{label} leaked keyHash")
print(f"verified {label} does not expose generated key material")
PY
}

assert_admin_audit_actions() {
  RESPONSE_BODY="$HTTP_BODY" CREATED_KEY="$MANAGED_KEY" ROTATED_KEY="$ROTATED_KEY" python3 - <<'PY'
import json
import os

body = os.environ["RESPONSE_BODY"]
created_key = os.environ["CREATED_KEY"]
rotated_key = os.environ["ROTATED_KEY"]
if created_key in body or rotated_key in body or "keyHash" in body:
    raise SystemExit("admin identity audit leaked generated key material")
rows = json.loads(body)["data"]
actions = [row.get("action") for row in rows]
expected = ["admin_identity.created", "admin_identity.key_rotated", "admin_identity.disabled"]
if actions != expected:
    raise SystemExit(f"expected admin identity audit actions {expected}, got {actions}")
print("verified admin identity lifecycle audit actions")
PY
}

need curl
need go
need python3

if port_in_use "$API_PORT"; then
  echo "API port $API_PORT is already in use" >&2
  exit 1
fi

mkdir -p "$LOG_DIR"
cd "$ROOT_DIR"

echo "AgentHarbor admin access management scenario"
echo "BASE_URL=$BASE_URL"
echo "RUN_ID=$RUN_ID"
echo "AGENT_HARBOR_ADMIN_IDENTITIES=platform=provided|role=platform_admin"

AGENT_HARBOR_ADDR="$API_ADDR" \
AGENT_HARBOR_ADMIN_IDENTITIES="platform=${PLATFORM_KEY}|role=platform_admin" \
AGENT_HARBOR_SESSION_SECRET="admin-access-session-secret-${RUN_ID}" \
AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=false \
AGENT_HARBOR_DATABASE_URL= \
AGENT_HARBOR_CREDENTIAL_KEY= \
  go run ./cmd/agent-harbor > "$LOG_DIR/api.log" 2>&1 &
PIDS+=("$!")

wait_http "API" "$BASE_URL/healthz"

request GET "/healthz" none
expect_status 200 "health check"

request POST "/api/v1/admin-identities" platform "$(json_body admin "platform" "Duplicate Bootstrap Actor" "$EAST_TENANT_ID" "$EAST_WORKSPACE_ID")"
expect_status 400 "managed admin actor cannot reuse bootstrap actor"
if [[ "$HTTP_BODY" != *"actor already exists"* ]]; then
  echo "expected duplicate bootstrap actor rejection to explain actor collision" >&2
  echo "$HTTP_BODY" >&2
  exit 1
fi
echo "verified managed admin actor cannot reuse bootstrap actor"

request POST "/api/v1/admin-identities" platform "$(json_body admin "$MANAGED_ACTOR" "Managed East Administrator" "$EAST_TENANT_ID" "$EAST_WORKSPACE_ID")"
expect_status 201 "create managed tenant admin"
MANAGED_ID="$(json_get data.identity.id)"
MANAGED_KEY="$(json_get data.key)"
echo "created managed admin identity: $MANAGED_ID"

request_managed_login_session "$MANAGED_KEY"
expect_status 200 "managed tenant admin login"
assert_login_session "$MANAGED_ACTOR" "tenant_admin" "$EAST_TENANT_ID" "$EAST_WORKSPACE_ID"

request GET "/api/v1/admin-identities" managed
expect_status 403 "tenant admin cannot manage administrator identities"
echo "verified tenant admin cannot call administrator management API"

request POST "/api/v1/tenants" platform "$(json_body tenant "$ROOT_TENANT_ID" "" "Admin Access Root")"
expect_status 201 "create root tenant"
request POST "/api/v1/tenants" platform "$(json_body tenant "$EAST_TENANT_ID" "$ROOT_TENANT_ID" "Admin Access East")"
expect_status 201 "create east tenant"
request POST "/api/v1/tenants" platform "$(json_body tenant "$WEST_TENANT_ID" "$ROOT_TENANT_ID" "Admin Access West")"
expect_status 201 "create west tenant"

request POST "/api/v1/agents" platform "$(json_body agent "$EAST_TENANT_ID" "$EAST_WORKSPACE_ID" "East platform seeded agent")"
expect_status 201 "create platform east agent"
request POST "/api/v1/agents" platform "$(json_body agent "$WEST_TENANT_ID" "$WEST_WORKSPACE_ID" "West platform seeded agent")"
expect_status 201 "create platform west agent"

request GET "/api/v1/agents" managed
expect_status 200 "managed tenant admin list scoped agents"
assert_agents_scope 1 "$EAST_TENANT_ID" "$EAST_WORKSPACE_ID"

request POST "/api/v1/agents" managed "$(json_body agent "$EAST_TENANT_ID" "$EAST_WORKSPACE_ID" "East managed admin created agent")"
expect_status 201 "managed tenant admin create in-scope agent"

request POST "/api/v1/agents" managed "$(json_body agent "$WEST_TENANT_ID" "$WEST_WORKSPACE_ID" "West managed admin denied agent")"
expect_status 403 "managed tenant admin create west agent"

request GET "/api/v1/agents?tenantId=${WEST_TENANT_ID}&workspaceId=${WEST_WORKSPACE_ID}" managed
expect_status 403 "managed tenant admin blocked from west query"

request GET "/api/v1/agents" managed
expect_status 200 "managed tenant admin list scoped agents after create"
assert_agents_scope 2 "$EAST_TENANT_ID" "$EAST_WORKSPACE_ID"

request GET "/api/v1/admin-identities" platform
expect_status 200 "platform list administrator identities"
assert_body_excludes_secret "admin identity list" "$MANAGED_KEY"

request POST "/api/v1/admin-identities/${MANAGED_ID}/key:rotate" platform
expect_status 200 "rotate managed admin key"
ROTATED_KEY="$(json_get data.key)"
if [[ -z "$ROTATED_KEY" || "$ROTATED_KEY" == "$MANAGED_KEY" ]]; then
  echo "expected rotated key to be non-empty and different" >&2
  exit 1
fi

request POST "/api/v1/auth/login" none "$(json_body login "$MANAGED_KEY")"
expect_status 401 "old managed admin key rejected after rotation"

request_with_managed_session POST "/api/v1/agents" "$(json_body agent "$EAST_TENANT_ID" "$EAST_WORKSPACE_ID" "Stale managed session denied agent")"
expect_status 401 "old managed admin session rejected after rotation"
echo "verified managed admin key rotation invalidates existing browser sessions"

request_managed_login_session "$ROTATED_KEY"
expect_status 200 "rotated managed admin key login"
assert_login_session "$MANAGED_ACTOR" "tenant_admin" "$EAST_TENANT_ID" "$EAST_WORKSPACE_ID"

request GET "/api/v1/admin-identities" platform
expect_status 200 "platform list administrator identities after rotation"
assert_body_excludes_secret "admin identity list after rotation" "$MANAGED_KEY"
assert_body_excludes_secret "admin identity list after rotation" "$ROTATED_KEY"

request POST "/api/v1/admin-identities/${MANAGED_ID}:disable" platform
expect_status 200 "disable managed admin identity"

request POST "/api/v1/auth/login" none "$(json_body login "$ROTATED_KEY")"
expect_status 401 "disabled managed admin key rejected"

request_with_managed_session POST "/api/v1/agents" "$(json_body agent "$EAST_TENANT_ID" "$EAST_WORKSPACE_ID" "Disabled managed session denied agent")"
expect_status 401 "disabled managed admin session rejected"
echo "verified managed admin disable invalidates existing browser sessions"

request POST "/api/v1/admin-identities/${MANAGED_ID}/key:rotate" platform
expect_status 409 "disabled managed admin key rotation rejected"
assert_body_contains "ADMIN_IDENTITY_DISABLED" "disabled managed admin key rotation"
echo "verified disabled managed admin identities cannot rotate keys"

request GET "/api/v1/audit/events?resourceType=admin_identity" platform
expect_status 200 "admin identity audit events"
assert_admin_audit_actions

echo "admin access management scenario complete"
