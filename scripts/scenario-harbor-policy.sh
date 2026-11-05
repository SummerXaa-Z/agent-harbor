#!/usr/bin/env bash
# M1 policy-mode end to end: the harbor import -> policy boot -> enforce ->
# explain loop against a real PostgreSQL-backed server. Skips (exit 0)
# when no PostgreSQL is reachable, like the store integration tests.
set -euo pipefail

API_PORT="${AGENT_HARBOR_POLICY_API_PORT:-9197}"
BASE_URL="http://127.0.0.1:${API_PORT}"
MOCK_PORT="${AGENT_HARBOR_POLICY_MOCK_PORT:-8789}"
RUN_ID="policy-$(date +%Y%m%d%H%M%S)"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
LOG_DIR="${TMPDIR:-/tmp}/agent-harbor-policy-${RUN_ID}"
PIDS=()
API_PID=""
HTTP_STATUS=""
HTTP_BODY=""

DATABASE_URL="${AGENT_HARBOR_POLICY_DATABASE_URL:-${AGENT_HARBOR_TEST_DATABASE_URL:-}}"
DB_NAME="agent_harbor_policy_$(date +%Y%m%d%H%M%S)"

PSQL_BIN="$(command -v psql || true)"
if [[ -z "$PSQL_BIN" && -x "$HOME/.homebrew/opt/postgresql@16/bin/psql" ]]; then
  PSQL_BIN="$HOME/.homebrew/opt/postgresql@16/bin/psql"
fi

cleanup() {
  for pid in "${PIDS[@]:-}"; do
    kill "$pid" >/dev/null 2>&1 || true
  done
  if [[ -n "$DATABASE_URL" && -n "$PSQL_BIN" ]]; then
    "$PSQL_BIN" "$DATABASE_URL" -c "drop database if exists $DB_NAME" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

json_get() {
  python3 -c 'import json,sys; document=json.load(sys.stdin); '"$1"
}

request() {
  local method="$1" path="$2" body="${3:-}" bearer="${4:-}" tmp
  tmp="$(mktemp)"
  local args=(-sS -o "$tmp" -w "%{http_code}" -X "$method" "$BASE_URL$path" -H "Content-Type: application/json")
  if [[ -n "$bearer" ]]; then args+=(-H "Authorization: Bearer $bearer"); fi
  if [[ -n "$body" ]]; then args+=(--data "$body"); fi
  HTTP_STATUS="$(curl "${args[@]}")"
  HTTP_BODY="$(python3 -c "print(open('$tmp').read())")"
  rm -f "$tmp"
}

expect_status() {
  if [[ "$HTTP_STATUS" != "$1" ]]; then
    echo "expected status $1, got $HTTP_STATUS: $HTTP_BODY" >&2
    exit 1
  fi
}

expect_body() {
  if [[ "$HTTP_BODY" != *"$1"* ]]; then
    echo "expected body to contain: $1" >&2
    echo "body: $HTTP_BODY" >&2
    exit 1
  fi
}

wait_http() {
  local i
  for i in $(seq 1 40); do
    if curl -sS -o /dev/null --max-time 1 "$BASE_URL/healthz" 2>/dev/null; then return 0; fi
    sleep 1
  done
  echo "server did not become healthy" >&2
  exit 1
}

if [[ -z "$DATABASE_URL" || -z "$PSQL_BIN" ]]; then
  echo "harbor policy scenario: skipping (set AGENT_HARBOR_TEST_DATABASE_URL and provide psql to run)"
  exit 0
fi

mkdir -p "$LOG_DIR"
cd "$ROOT_DIR"

echo "harbor policy scenario: RUN_ID=$RUN_ID DB=$DB_NAME"
"$PSQL_BIN" "$DATABASE_URL" -c "drop database if exists $DB_NAME" >/dev/null
"$PSQL_BIN" "$DATABASE_URL" -c "create database $DB_NAME" >/dev/null

CREDENTIAL_KEY="$(python3 -c 'import base64,os; print(base64.b64encode(os.urandom(32)).decode())')"
go build -o "$LOG_DIR/api-server" ./cmd/agent-harbor
go build -o "$LOG_DIR/harbor" ./cmd/harbor

python3 - "$MOCK_PORT" > "$LOG_DIR/mock.log" 2>&1 <<'PY' &
import runpy, sys
sys.argv = ["mock-mcp-server.py", "--host", "127.0.0.1", "--port", sys.argv[1]]
runpy.run_path("scripts/mock-mcp-server.py", run_name="__main__")
PY
PIDS+=("$!")

cat > "$LOG_DIR/servers.json" <<JSON
{"mcpServers": {"mocktools": {"type": "http", "url": "http://127.0.0.1:${MOCK_PORT}/mcp"}}}
JSON

start_api() {
  AGENT_HARBOR_ADDR="127.0.0.1:${API_PORT}" \
  AGENT_HARBOR_DATABASE_URL="${DATABASE_URL%/*}/$DB_NAME" \
  AGENT_HARBOR_CREDENTIAL_KEY="$CREDENTIAL_KEY" \
  AGENT_HARBOR_POLICY_FILE="${1:-}" \
  AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true \
  AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true \
    "$LOG_DIR/api-server" >> "$LOG_DIR/api.log" 2>&1 &
  API_PID="$!"
  PIDS+=("$API_PID")
  wait_http
}

# Phase 1: no policy - the adoption funnel runs against a plain server.
start_api
"$LOG_DIR/harbor" import --from "$LOG_DIR/servers.json" --url "$BASE_URL" \
  --output "$LOG_DIR/harbor.yaml" --state "$LOG_DIR/state.json" > "$LOG_DIR/import.log"
AGENT_KEY="$(grep -m1 '^  key ' "$LOG_DIR/import.log" | awk '{print $2}')"
if [[ "$AGENT_KEY" != ah_* ]]; then
  echo "import did not print an agent key:" >&2; grep -A2 'connect claude-code' "$LOG_DIR/import.log" >&2; exit 1
fi
TARGET_ID="$(python3 -c "import json,sys; print(json.load(open(sys.argv[1]))['targets']['mocktools'])" "$LOG_DIR/state.json")"
CALLER_ID="$(python3 -c "import json,sys; print(json.load(open(sys.argv[1]))['clients']['claude-code'])" "$LOG_DIR/state.json")"

request POST "/api/v1/targets/$TARGET_ID/capabilities:refresh" '{}'
expect_status 200 "refresh capabilities"
expect_body "search_customer"

# Tighten the baseline: deny exports for the imported caller.
python3 - "$LOG_DIR/harbor.yaml" "$CALLER_ID" "$TARGET_ID" <<'PY'
import sys
path, caller, target = sys.argv[1:4]
text = open(path).read()
anchor = f"  - id: {caller}\n    workspace: main\n"
assert anchor in text, "caller block missing"
text = text.replace(anchor, anchor + f"    deny:\n      - capability: \"{target}.export_*\"\n        reason: exports stay manual\n", 1)
open(path, "w").write(text)
PY

# Phase 2: same database, now with the policy loaded.
kill "$API_PID" >/dev/null 2>&1 || true
# Wait for the port to be released: healthz on the old process would
# otherwise satisfy phase two's wait against the no-policy server.
for i in $(seq 1 30); do
  if ! lsof -nP -iTCP:"$API_PORT" -sTCP:LISTEN >/dev/null 2>&1; then break; fi
  sleep 0.5
done
start_api "$LOG_DIR/harbor.yaml"

request GET "/api/v1/system/info"
expect_status 200 "system info"
if [[ "$(printf '%s' "$HTTP_BODY" | json_get 'print(document["data"]["policyMode"])')" != "True" ]]; then
  echo "policyMode should be true with a policy file loaded" >&2
  exit 1
fi

request POST "/api/v1/mcp/agents/$TARGET_ID/rpc" '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search_customer","arguments":{"q":"A-1"}}}' "$AGENT_KEY"
expect_status 200 "allowed call proxied upstream"
expect_body "mock result"

request POST "/api/v1/mcp/agents/$TARGET_ID/rpc" '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"export_contracts","arguments":{}}}' "$AGENT_KEY"
expect_status 403 "policy denial"
expect_body "Do not retry"
expect_body "harbor.yaml:"
AUDIT_ID="$(printf '%s' "$HTTP_BODY" | json_get 'print(document["details"]["auditId"])')"

request GET "/api/v1/audit/traces?highRisk=true&limit=50"
expect_status 200 "high-risk trace filter"
HIGH_RISK_COUNT="$(printf '%s' "$HTTP_BODY" | json_get 'print(len(document["data"]))')"
if [[ "$HIGH_RISK_COUNT" -lt 1 ]]; then
  echo "high-risk filter should return the flagged denial" >&2
  exit 1
fi

"$LOG_DIR/harbor" why --url "$BASE_URL" "$AUDIT_ID" > "$LOG_DIR/why.log"
if [[ "$?" -ne 0 ]] || ! grep -q "harbor.yaml:" "$LOG_DIR/why.log"; then
  echo "harbor why should explain the denial with its rule reference:" >&2
  grep -v '^  key' "$LOG_DIR/why.log" >&2 || true
  exit 1
fi

echo "harbor policy scenario passed: import -> policy boot -> allow/deny -> explain -> audit"
