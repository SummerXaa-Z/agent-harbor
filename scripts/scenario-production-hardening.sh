#!/usr/bin/env bash
set -euo pipefail

API_HOST="${AGENT_HARBOR_PRODUCTION_GATE_API_HOST:-127.0.0.1}"
API_PORT="${AGENT_HARBOR_PRODUCTION_GATE_API_PORT:-9091}"
UNAUTH_API_PORT="${AGENT_HARBOR_PRODUCTION_GATE_UNAUTH_API_PORT:-$((API_PORT + 1))}"
API_ADDR="${AGENT_HARBOR_PRODUCTION_GATE_API_ADDR:-${API_HOST}:${API_PORT}}"
BASE_URL="${AGENT_HARBOR_PRODUCTION_GATE_BASE_URL:-http://${API_HOST}:${API_PORT}}"
UNAUTH_BASE_URL="${AGENT_HARBOR_PRODUCTION_GATE_UNAUTH_BASE_URL:-http://${API_HOST}:${UNAUTH_API_PORT}}"
ADMIN_KEY="${ADMIN_KEY:-production-hardening-admin-key}"
WRONG_ADMIN_KEY="${WRONG_ADMIN_KEY:-production-hardening-wrong-key}"
RUN_ID="${RUN_ID:-production-hardening-$(date +%Y%m%d%H%M%S)}"

if [[ "$WRONG_ADMIN_KEY" == "$ADMIN_KEY" ]]; then
  WRONG_ADMIN_KEY="${ADMIN_KEY}-wrong"
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOG_DIR="${TMPDIR:-/tmp}/agent-harbor-production-hardening-${RUN_ID}"
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
  local auth_mode="${3:-none}"
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
    none)
      ;;
    wrong)
      args+=(-H "X-Admin-Key: $WRONG_ADMIN_KEY")
      ;;
    admin)
      args+=(-H "X-Admin-Key: $ADMIN_KEY")
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

assert_trusted_forwarded_login_sets_secure_cookie() {
	local headers
	local body
	local status
	headers="$(mktemp)"
	body="$(mktemp)"
	if ! status="$(curl -sS -D "$headers" -o "$body" -w "%{http_code}" \
		-X POST "$BASE_URL/api/v1/auth/login" \
		-H "Content-Type: application/json" \
		-H "X-Forwarded-Proto: https" \
		-d "{\"adminKey\":\"$ADMIN_KEY\"}")"; then
		rm -f "$headers" "$body"
		echo "curl failed for trusted forwarded console login" >&2
		show_logs
		exit 1
	fi
	if [[ "$status" != "200" ]]; then
		echo "expected trusted forwarded console login status 200, got $status" >&2
		cat "$body" >&2
		rm -f "$headers" "$body"
		show_logs
		exit 1
	fi
	if ! tr -d '\r' < "$headers" | grep -qi '^Set-Cookie: agent_harbor_session=.*; Secure'; then
		echo "trusted forwarded console login did not set a Secure session cookie" >&2
		cat "$headers" >&2
		rm -f "$headers" "$body"
		show_logs
		exit 1
	fi
	rm -f "$headers" "$body"
}

json_body() {
  python3 - "$@" <<'PY'
import json
import sys

kind = sys.argv[1]
run_id = sys.argv[2]
if kind == "local-agent":
    body = {
        "name": f"Production Hardening Local Agent {run_id}",
        "workspaceId": "ws-production-hardening",
        "channelType": "local",
        "status": "active",
    }
elif kind == "loopback-mcp":
    body = {
        "name": f"Production Hardening Blocked Loopback MCP {run_id}",
        "workspaceId": "ws-production-hardening",
        "channelType": "mcp",
        "status": "active",
        "channelConfig": {"endpoint": "http://127.0.0.1:8787/mcp"},
    }
elif kind == "public-mcp":
    body = {
        "name": f"Production Hardening Public MCP {run_id}",
        "workspaceId": "ws-production-hardening",
        "channelType": "mcp",
        "status": "active",
        "channelConfig": {"endpoint": "https://api.example.com/mcp"},
    }
elif kind == "management-tools":
    body = {
        "jsonrpc": "2.0",
        "id": f"production-hardening-tools-{run_id}",
        "method": "tools/list",
        "params": {},
    }
else:
    raise SystemExit(f"unknown body kind: {kind}")
print(json.dumps(body, separators=(",", ":")))
PY
}

need curl
need go
need python3

if port_in_use "$API_PORT"; then
	echo "API port $API_PORT is already in use" >&2
	exit 1
fi
if port_in_use "$UNAUTH_API_PORT"; then
	echo "unauthenticated admin check API port $UNAUTH_API_PORT is already in use" >&2
	exit 1
fi
if port_in_use "$((UNAUTH_API_PORT + 1))"; then
	echo "production unsafe config check API port $((UNAUTH_API_PORT + 1)) is already in use" >&2
	exit 1
fi
if port_in_use "$((UNAUTH_API_PORT + 2))"; then
	echo "production weak session secret check API port $((UNAUTH_API_PORT + 2)) is already in use" >&2
	exit 1
fi
if port_in_use "$((UNAUTH_API_PORT + 3))"; then
	echo "production invalid CORS check API port $((UNAUTH_API_PORT + 3)) is already in use" >&2
	exit 1
fi
if port_in_use "$((UNAUTH_API_PORT + 4))"; then
	echo "production missing database check API port $((UNAUTH_API_PORT + 4)) is already in use" >&2
	exit 1
fi
if port_in_use "$((UNAUTH_API_PORT + 5))"; then
	echo "production preflight before storage check API port $((UNAUTH_API_PORT + 5)) is already in use" >&2
	exit 1
fi
if port_in_use "$((UNAUTH_API_PORT + 6))"; then
	echo "production admin key conflict check API port $((UNAUTH_API_PORT + 6)) is already in use" >&2
	exit 1
fi
if port_in_use "$((UNAUTH_API_PORT + 7))"; then
	echo "production missing session secret check API port $((UNAUTH_API_PORT + 7)) is already in use" >&2
	exit 1
fi

mkdir -p "$LOG_DIR"
cd "$ROOT_DIR"

echo "AgentHarbor production hardening gate"
echo "BASE_URL=$BASE_URL"
echo "RUN_ID=$RUN_ID"
echo "ADMIN_KEY=provided"
echo "AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=false"

AGENT_HARBOR_ADDR="${API_HOST}:${UNAUTH_API_PORT}" \
AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=false \
AGENT_HARBOR_DATABASE_URL= \
AGENT_HARBOR_CREDENTIAL_KEY= \
  go run ./cmd/agent-harbor > "$LOG_DIR/api-unauth.log" 2>&1 &
UNAUTH_PID="$!"
PIDS+=("$UNAUTH_PID")

wait_http "Unauthenticated admin check API" "$UNAUTH_BASE_URL/healthz"

tmp="$(mktemp)"
if ! unauth_status="$(curl -sS -o "$tmp" -w "%{http_code}" -X POST "$UNAUTH_BASE_URL/api/v1/agents" -H "Content-Type: application/json" -d "$(json_body local-agent "$RUN_ID-unauth")")"; then
	rm -f "$tmp"
	echo "curl failed for unauthenticated admin default check" >&2
	show_logs
	exit 1
fi
unauth_body="$(<"$tmp")"
rm -f "$tmp"
if [[ "$unauth_status" != "401" || "$unauth_body" != *"admin authentication is required"* ]]; then
	echo "expected fail-closed admin default status 401, got $unauth_status" >&2
	echo "$unauth_body" >&2
	show_logs
	exit 1
fi
echo "management endpoint fails closed without configured admin authentication"
kill "$UNAUTH_PID" >/dev/null 2>&1 || true
wait "$UNAUTH_PID" >/dev/null 2>&1 || true

if AGENT_HARBOR_ADDR="${API_HOST}:$((UNAUTH_API_PORT + 1))" \
	AGENT_HARBOR_DEPLOYMENT_MODE=production \
	AGENT_HARBOR_ADMIN_KEY="$ADMIN_KEY" \
	AGENT_HARBOR_SESSION_SECRET=production-hardening-session-secret \
	AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true \
	AGENT_HARBOR_DATABASE_URL= \
	AGENT_HARBOR_CREDENTIAL_KEY= \
	go run ./cmd/agent-harbor > "$LOG_DIR/api-prod-unsafe.log" 2>&1; then
	echo "expected production mode to reject AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true" >&2
	show_logs
	exit 1
fi
if ! grep -q "AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN" "$LOG_DIR/api-prod-unsafe.log"; then
	echo "production unsafe flag failure did not mention AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN" >&2
	show_logs
	exit 1
fi
echo "production deployment preflight rejects unauthenticated admin flag"

AGENT_HARBOR_ADDR="${API_HOST}:$((UNAUTH_API_PORT + 7))" \
AGENT_HARBOR_DEPLOYMENT_MODE=production \
AGENT_HARBOR_ADMIN_KEY="$ADMIN_KEY" \
AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=false \
AGENT_HARBOR_DATABASE_URL= \
AGENT_HARBOR_CREDENTIAL_KEY= \
	go run ./cmd/agent-harbor > "$LOG_DIR/api-prod-missing-session-secret.log" 2>&1 && {
	echo "expected production mode to require AGENT_HARBOR_SESSION_SECRET" >&2
	show_logs
	exit 1
}
if ! grep -q "AGENT_HARBOR_SESSION_SECRET" "$LOG_DIR/api-prod-missing-session-secret.log"; then
	echo "production missing session secret failure did not mention AGENT_HARBOR_SESSION_SECRET" >&2
	show_logs
	exit 1
fi
echo "production deployment preflight requires explicit session secret"

AGENT_HARBOR_ADDR="${API_HOST}:$((UNAUTH_API_PORT + 2))" \
AGENT_HARBOR_DEPLOYMENT_MODE=production \
AGENT_HARBOR_ADMIN_KEY="$ADMIN_KEY" \
AGENT_HARBOR_SESSION_SECRET=secret \
AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=false \
AGENT_HARBOR_DATABASE_URL= \
AGENT_HARBOR_CREDENTIAL_KEY= \
	go run ./cmd/agent-harbor > "$LOG_DIR/api-prod-weak-session-secret.log" 2>&1 && {
	echo "expected production mode to reject weak AGENT_HARBOR_SESSION_SECRET" >&2
	show_logs
	exit 1
}
if ! grep -q "AGENT_HARBOR_SESSION_SECRET" "$LOG_DIR/api-prod-weak-session-secret.log"; then
	echo "production weak session secret failure did not mention AGENT_HARBOR_SESSION_SECRET" >&2
	show_logs
	exit 1
fi
echo "production deployment preflight rejects weak session secret"

AGENT_HARBOR_ADDR="${API_HOST}:$((UNAUTH_API_PORT + 3))" \
AGENT_HARBOR_DEPLOYMENT_MODE=production \
AGENT_HARBOR_ADMIN_KEY="$ADMIN_KEY" \
AGENT_HARBOR_SESSION_SECRET=production-hardening-session-secret \
AGENT_HARBOR_CORS_ORIGINS="*" \
AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=false \
AGENT_HARBOR_DATABASE_URL= \
AGENT_HARBOR_CREDENTIAL_KEY= \
	go run ./cmd/agent-harbor > "$LOG_DIR/api-prod-invalid-cors.log" 2>&1 && {
	echo "expected production mode to reject invalid AGENT_HARBOR_CORS_ORIGINS" >&2
	show_logs
	exit 1
}
if ! grep -q "AGENT_HARBOR_CORS_ORIGINS" "$LOG_DIR/api-prod-invalid-cors.log"; then
	echo "production invalid CORS failure did not mention AGENT_HARBOR_CORS_ORIGINS" >&2
	show_logs
	exit 1
fi
echo "production deployment preflight rejects invalid CORS origins"

AGENT_HARBOR_ADDR="${API_HOST}:$((UNAUTH_API_PORT + 6))" \
AGENT_HARBOR_DEPLOYMENT_MODE=production \
AGENT_HARBOR_ADMIN_KEY="$ADMIN_KEY" \
AGENT_HARBOR_ADMIN_IDENTITIES="platform=${ADMIN_KEY}|role=platform_admin" \
AGENT_HARBOR_SESSION_SECRET=production-hardening-session-secret \
AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=false \
AGENT_HARBOR_DATABASE_URL= \
AGENT_HARBOR_CREDENTIAL_KEY= \
	go run ./cmd/agent-harbor > "$LOG_DIR/api-prod-admin-key-conflict.log" 2>&1 && {
	echo "expected production mode to reject shared admin key matching a named identity key" >&2
	show_logs
	exit 1
}
if ! grep -q "AGENT_HARBOR_ADMIN_KEY" "$LOG_DIR/api-prod-admin-key-conflict.log" || ! grep -q "AGENT_HARBOR_ADMIN_IDENTITIES" "$LOG_DIR/api-prod-admin-key-conflict.log"; then
	echo "production admin key conflict failure did not mention both admin key settings" >&2
	show_logs
	exit 1
fi
if grep -q "$ADMIN_KEY" "$LOG_DIR/api-prod-admin-key-conflict.log"; then
	echo "production admin key conflict failure leaked the configured admin key" >&2
	show_logs
	exit 1
fi
echo "production deployment preflight rejects shared admin key matching named identity"

AGENT_HARBOR_ADDR="${API_HOST}:$((UNAUTH_API_PORT + 5))" \
AGENT_HARBOR_DEPLOYMENT_MODE=production \
AGENT_HARBOR_ADMIN_KEY=short-key \
AGENT_HARBOR_SESSION_SECRET=production-hardening-session-secret \
AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=false \
AGENT_HARBOR_DATABASE_URL="://invalid-postgres-url" \
AGENT_HARBOR_CREDENTIAL_KEY= \
	go run ./cmd/agent-harbor > "$LOG_DIR/api-prod-preflight-before-storage.log" 2>&1 && {
	echo "expected production preflight to reject weak admin key before storage initialization" >&2
	show_logs
	exit 1
}
if ! grep -q "AGENT_HARBOR_ADMIN_KEY" "$LOG_DIR/api-prod-preflight-before-storage.log"; then
	echo "production preflight-before-storage failure did not mention AGENT_HARBOR_ADMIN_KEY" >&2
	show_logs
	exit 1
fi
if grep -q "connect postgres" "$LOG_DIR/api-prod-preflight-before-storage.log"; then
	echo "production preflight-before-storage check attempted PostgreSQL before failing config preflight" >&2
	show_logs
	exit 1
fi
echo "production deployment preflight runs before storage initialization"

AGENT_HARBOR_ADDR="${API_HOST}:$((UNAUTH_API_PORT + 4))" \
AGENT_HARBOR_DEPLOYMENT_MODE=production \
AGENT_HARBOR_ADMIN_KEY="$ADMIN_KEY" \
AGENT_HARBOR_SESSION_SECRET=production-hardening-session-secret \
AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=false \
AGENT_HARBOR_DATABASE_URL= \
AGENT_HARBOR_CREDENTIAL_KEY= \
	go run ./cmd/agent-harbor > "$LOG_DIR/api-prod-missing-database.log" 2>&1 && {
	echo "expected production mode to require AGENT_HARBOR_DATABASE_URL" >&2
	show_logs
	exit 1
}
if ! grep -q "AGENT_HARBOR_DATABASE_URL" "$LOG_DIR/api-prod-missing-database.log"; then
	echo "production missing database failure did not mention AGENT_HARBOR_DATABASE_URL" >&2
	show_logs
	exit 1
fi
echo "production deployment preflight requires persistent database storage"

AGENT_HARBOR_ADDR="$API_ADDR" \
AGENT_HARBOR_ADMIN_KEY="$ADMIN_KEY" \
AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=false \
AGENT_HARBOR_DATABASE_URL= \
AGENT_HARBOR_CREDENTIAL_KEY= \
  go run ./cmd/agent-harbor > "$LOG_DIR/api.log" 2>&1 &
PIDS+=("$!")

wait_http "API" "$BASE_URL/healthz"

request GET "/healthz" none
expect_status 200 "public health check"
echo "public health check verified"

assert_trusted_forwarded_login_sets_secure_cookie
echo "trusted forwarded HTTPS console login sets Secure session cookie"

request POST "/api/v1/agents" none "$(json_body local-agent "$RUN_ID")"
expect_status 401 "management endpoint without admin key"
assert_body_contains "missing or invalid admin key" "management endpoint without admin key"
echo "management endpoint rejects missing admin key"

request GET "/api/v1/agents" wrong
expect_status 401 "management endpoint with wrong admin key"
assert_body_contains "missing or invalid admin key" "management endpoint with wrong admin key"
echo "management endpoint rejects wrong admin key"

request GET "/api/v1/agents" admin
expect_status 200 "management endpoint with admin key"
echo "management endpoint accepts configured admin key"

request GET "/api/v1/permission-packages/templates" none
expect_status 401 "permission package templates without admin key"
echo "permission package APIs require admin key"

request GET "/api/v1/permission-packages/templates" admin
expect_status 200 "permission package templates with admin key"
echo "permission package APIs accept configured admin key"

request POST "/api/v1/management/mcp" none "$(json_body management-tools "$RUN_ID")"
expect_status 401 "management MCP without admin key"
echo "management MCP requires admin key"

request POST "/api/v1/management/mcp" admin "$(json_body management-tools "$RUN_ID")"
expect_status 200 "management MCP with admin key"
assert_body_contains "list_permission_package_templates" "management MCP tools/list"
echo "management MCP accepts configured admin key"

request POST "/api/v1/agents" admin "$(json_body loopback-mcp "$RUN_ID")"
expect_status 400 "loopback MCP endpoint with private upstreams disabled"
assert_body_contains "endpoint host is not allowed" "loopback MCP endpoint"
echo "private and loopback upstreams are rejected by default"

request POST "/api/v1/agents" admin "$(json_body public-mcp "$RUN_ID")"
expect_status 201 "public MCP endpoint with admin key"
echo "public HTTPS MCP endpoint remains registrable"

echo "production hardening gate complete"
