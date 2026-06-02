#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
cd "$ROOT_DIR"

BASE_URL="${BASE_URL:-http://127.0.0.1:9090}"
ADMIN_KEY="${ADMIN_KEY:-}"
export BASE_URL ADMIN_KEY

scripts=(
  scripts/scenario-governance-loop.sh
  scripts/scenario-registry-cleanup.sh
  scripts/scenario-mcp-policy.sh
  scripts/scenario-credential-redaction.sh
  scripts/scenario-retry-config.sh
  scripts/scenario-runtime-metrics.sh
  scripts/scenario-credential-rotation.sh
  scripts/scenario-management-audit.sh
  scripts/scenario-route-policies.sh
  scripts/scenario-route-policy-retry.sh
  scripts/scenario-transactional-audit.sh
  scripts/scenario-mcp-capability-governance.sh
  scripts/scenario-data-permission-enforcement.sh
  scripts/scenario-tenant-hierarchy.sh
  scripts/scenario-tenant-access-profile.sh
)

echo "AgentHarbor full scenario suite"
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
echo "all scenarios complete"
