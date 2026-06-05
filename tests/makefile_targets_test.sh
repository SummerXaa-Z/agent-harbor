#!/usr/bin/env bash
set -euo pipefail

assert_target_depends_on() {
  local target="$1"
  local dependency="$2"
  if ! awk -v target="$target" -v dependency="$dependency" '
    $0 ~ "^[[:space:]]*" target ":" {
      split($0, parts, ":")
      split(parts[2], deps, /[[:space:]]+/)
      for (i in deps) {
        if (deps[i] == dependency) {
          found = 1
        }
      }
    }
    END { exit found ? 0 : 1 }
  ' Makefile; then
    echo "expected target ${target} to depend on ${dependency}" >&2
    exit 1
  fi
}

assert_target_depends_on "frontend-test" "frontend-deps"
assert_target_depends_on "frontend-build" "frontend-deps"
assert_target_depends_on "demo" "frontend-deps"
assert_target_depends_on "ai-admin-browser-journey" "frontend-deps"
