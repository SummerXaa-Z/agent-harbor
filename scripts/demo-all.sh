#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
cd "$ROOT_DIR"

BASE_URL="${BASE_URL:-http://127.0.0.1:9090}"
ADMIN_KEY="${ADMIN_KEY:-}"
export BASE_URL ADMIN_KEY

scripts=(
  scripts/demo-governance-loop.sh
  scripts/demo-sprint2-cleanup.sh
  scripts/demo-sprint3-mcp-policy.sh
  scripts/demo-sprint4-credentials.sh
  scripts/demo-sprint5-retry-config.sh
  scripts/demo-sprint6-runtime-metrics.sh
  scripts/demo-sprint7-credential-rotation.sh
  scripts/demo-sprint8-management-audit.sh
  scripts/demo-sprint9-route-policies.sh
  scripts/demo-sprint10-route-policy-retry.sh
  scripts/demo-sprint11-transactional-audit.sh
)

echo "AgentHarbor full demo suite"
echo "BASE_URL=${BASE_URL}"
if [[ -n "$ADMIN_KEY" ]]; then
  echo "ADMIN_KEY=provided"
else
  echo "ADMIN_KEY=not set"
fi

for script in "${scripts[@]}"; do
  echo
  echo "==> ${script}"
  bash "$script"
done

echo
echo "all demos complete"
