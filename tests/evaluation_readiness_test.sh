#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
OUTPUT_DIR="${TMPDIR:-/tmp}/agent-harbor-evaluation-readiness-test-$$"
cleanup() {
  rm -rf "$OUTPUT_DIR"
}
trap cleanup EXIT

cd "$ROOT_DIR"
scripts/evaluation-readiness.sh --output-dir "$OUTPUT_DIR" >/tmp/agent-harbor-evaluation-readiness-test.log

assert_file_exists() {
  local file="$1"
  if [[ ! -f "$OUTPUT_DIR/$file" ]]; then
    echo "expected evaluator pack to contain $file" >&2
    exit 1
  fi
}

assert_file_contains() {
  local file="$1"
  local needle="$2"
  if ! grep -Fq "$needle" "$OUTPUT_DIR/$file"; then
    echo "expected $file to contain $needle" >&2
    exit 1
  fi
}

assert_file_exists "README.md"
assert_file_exists "fresh-run-checklist.md"
assert_file_exists "feedback-log.csv"
assert_file_exists "acceptance-report-notes.md"
assert_file_exists "evaluator-handoff.md"
assert_file_exists "environment-snapshot.md"

assert_file_contains "environment-snapshot.md" "Branch:"
assert_file_contains "environment-snapshot.md" "Commit:"
assert_file_contains "environment-snapshot.md" "Working tree:"
assert_file_contains "environment-snapshot.md" "Go:"
assert_file_contains "environment-snapshot.md" "Node:"
assert_file_contains "environment-snapshot.md" "pnpm:"
assert_file_contains "feedback-log.csv" "time-to-first-report"
assert_file_contains "acceptance-report-notes.md" "Report digest:"
