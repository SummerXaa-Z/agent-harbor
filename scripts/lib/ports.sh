#!/usr/bin/env bash

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

describe_port_owner() {
  local port="$1"
  if command -v lsof >/dev/null 2>&1; then
    echo "listener on TCP port ${port}:"
    lsof -nP -iTCP:"$port" -sTCP:LISTEN 2>/dev/null | sed 's/^/  /' || true
    return
  fi
  echo "listener details unavailable: lsof is not installed"
}

assert_port_free() {
  local label="$1"
  local port="$2"
  if port_in_use "$port"; then
    echo "$label port $port is already in use" >&2
    describe_port_owner "$port" >&2
    exit 1
  fi
}

kill_port_listener() {
  local port="$1"
  local signal="$2"
  local pid
  command -v lsof >/dev/null 2>&1 || return 0
  while IFS= read -r pid; do
    [[ -n "$pid" ]] || continue
    [[ "$pid" != "$$" ]] || continue
    kill "-$signal" "$pid" >/dev/null 2>&1 || true
  done < <(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)
}
