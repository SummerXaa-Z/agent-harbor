#!/usr/bin/env bash
set -euo pipefail

PINNED_PNPM_VERSION="${AGENT_HARBOR_PNPM_VERSION:-10.30.3}"
TARGET_DIR="."

for ((i = 1; i <= $#; i++)); do
  arg="${!i}"
  case "$arg" in
    --dir | -C)
      next=$((i + 1))
      if ((next <= $#)); then
        TARGET_DIR="${!next}"
      fi
      ;;
    --dir=*)
      TARGET_DIR="${arg#--dir=}"
      ;;
  esac
done

declare -a CANDIDATES=()

if [[ -n "${AGENT_HARBOR_PNPM_BIN:-}" ]]; then
  CANDIDATES+=("$AGENT_HARBOR_PNPM_BIN")
fi

while IFS= read -r candidate; do
  CANDIDATES+=("$candidate")
done < <(type -P -a pnpm 2>/dev/null || true)

SEEN=":"

for candidate in "${CANDIDATES[@]}"; do
  if [[ -z "$candidate" || ! -x "$candidate" || "$SEEN" == *":$candidate:"* ]]; then
    continue
  fi
  SEEN="${SEEN}${candidate}:"
  version="$("$candidate" --dir "$TARGET_DIR" --version 2>/dev/null | tail -n 1 || true)"
  if [[ "$version" == 10.* ]]; then
    exec "$candidate" "$@"
  fi
done

if command -v corepack >/dev/null 2>&1; then
  exec corepack "pnpm@${PINNED_PNPM_VERSION}" "$@"
fi

echo "No compatible pnpm 10.x executable found. Install pnpm ${PINNED_PNPM_VERSION} or set AGENT_HARBOR_PNPM_BIN." >&2
exit 127
