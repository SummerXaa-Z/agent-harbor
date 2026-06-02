#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:9090}"
ADMIN_KEY="${ADMIN_KEY:-}"
RUN_ID="${RUN_ID:-transactional-audit-$(date +%s)}"

admin_args=()
if [[ -n "$ADMIN_KEY" ]]; then
  admin_args=(-H "X-Admin-Key: ${ADMIN_KEY}")
fi

curl_json() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  if [[ -n "$body" ]]; then
    curl -fsS -X "$method" "${BASE_URL}${path}" "${admin_args[@]}" -H 'Content-Type: application/json' -d "$body"
  else
    curl -fsS -X "$method" "${BASE_URL}${path}" "${admin_args[@]}" -H 'Content-Type: application/json'
  fi
}

extract_data_field() {
  local json="$1"
  local field="$2"
  python3 - "$field" "$json" <<'PY'
import json
import sys
field = sys.argv[1]
payload = json.loads(sys.argv[2])
print(payload["data"][field])
PY
}

echo "== Transactional Audit transactional audit scenario (${RUN_ID}) =="

agent_payload=$(cat <<JSON
{
  "tenantId": "scenario-transactional-audit",
  "workspaceId": "ws-transactional-audit",
  "name": "Transactional Audit Audited Target",
  "channelType": "mcp",
  "channelConfig": {
    "endpoint": "https://api.example.com/mcp",
    "credentialHeaders": {
      "Authorization": "apiToken"
    }
  },
  "credentials": {
    "apiToken": "Bearer transactional-audit-secret"
  },
  "status": "active"
}
JSON
)

agent_resp="$(curl_json POST /api/v1/agents "$agent_payload")"
agent_id="$(extract_data_field "$agent_resp" id)"
echo "Created audited agent: ${agent_id}"

rotate_resp="$(curl_json POST "/api/v1/agents/${agent_id}/credentials:rotate" '{"credentials":{"apiToken":"Bearer transactional-audit-rotated-secret"}}')"
echo "Rotated credentials: $(extract_data_field "$rotate_resp" credentialVersion)"

events="$(curl_json GET "/api/v1/audit/events?tenantId=scenario-transactional-audit&workspaceId=ws-transactional-audit&resourceId=${agent_id}")"
python3 - "$events" <<'PY'
import json
import sys
payload = json.loads(sys.argv[1])
actions = [event["action"] for event in payload["data"]]
required = ["agent.created", "agent.credentials_rotated"]
missing = [action for action in required if action not in actions]
if missing:
    raise SystemExit(f"missing audit actions: {missing}; got {actions}")
text = json.dumps(payload)
if "transactional-audit-secret" in text or "transactional-audit-rotated-secret" in text:
    raise SystemExit("audit events leaked credential values")
print("Audit actions:", ", ".join(actions))
print("Credential values redacted from audit events")
PY
