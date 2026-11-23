#!/usr/bin/env python3
"""Run a disposable stdio MCP client, wrapper and audit server."""
import json
import os
from pathlib import Path
import secrets
import socket
import subprocess
import sys
import tempfile
import time
from urllib.error import URLError
from urllib.request import ProxyHandler, Request, build_opener


def sample_server():
    for line in sys.stdin:
        request = json.loads(line)
        if "id" not in request:
            continue
        method = request["method"]
        if method == "initialize":
            result = {
                "protocolVersion": request["params"]["protocolVersion"],
                "capabilities": {"tools": {}},
                "serverInfo": {"name": "sample", "version": "1"},
            }
        elif method == "tools/list":
            result = {"tools": [
                {"name": name, "description": "Return sample text", "inputSchema": {"type": "object"}}
                for name in ("hello", "goodbye")
            ]}
        elif method == "tools/call":
            result = {"content": [{"type": "text", "text": request["params"]["name"]}]}
        else:
            raise ValueError(f"unexpected method: {method}")
        print(json.dumps({"jsonrpc": "2.0", "id": request["id"], "result": result}), flush=True)


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def run_demo(root, work):
    started = time.monotonic()
    key = secrets.token_hex(24)
    env = {"PATH": os.environ["PATH"], "HOME": str(work)}
    for binary in ("harbor", "harbor-wrap", "agent-harbor"):
        subprocess.run(["go", "build", "-o", str(work / binary), f"./cmd/{binary}"],
                       cwd=root, check=True, timeout=240)
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        addr = f"127.0.0.1:{sock.getsockname()[1]}"
    url = "http://" + addr
    opener = build_opener(ProxyHandler({}))

    def api(path, payload=None):
        body = None if payload is None else json.dumps(payload).encode()
        headers = {"Content-Type": "application/json", "X-Admin-Key": key}
        with opener.open(Request(url + path, body, headers), timeout=2) as response:
            return json.load(response)

    def cli(*args):
        result = subprocess.run([str(work / "harbor"), *args], cwd=work,
                                env=dict(env, HARBOR_URL=url, HARBOR_ADMIN_KEY=key), text=True, capture_output=True, timeout=20)
        require(result.returncode == 0, result.stderr.strip())
        return result.stdout

    with subprocess.Popen([str(work / "agent-harbor")], cwd=work,
                          env=dict(env, AGENT_HARBOR_ADDR=addr, AGENT_HARBOR_ADMIN_KEY=key),
                          stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, text=True) as server:
        try:
            for _ in range(80):
                if server.poll() is not None:
                    raise RuntimeError("temporary audit server exited: " + server.stderr.read())
                try:
                    api("/healthz")
                    break
                except URLError:
                    time.sleep(0.1)
            else:
                raise RuntimeError("temporary audit server did not start")

            def register(name):
                return api("/api/v1/agents", {
                    "tenantId": "default", "workspaceId": "sample", "name": name,
                    "channelType": "local", "status": "active",
                })["data"]["id"]

            caller, target = register("Sample caller"), register("Sample target")
            policy = work / "harbor.yaml"
            policy.write_text(f"""version: 1
workspaces:
  - id: sample
    strict: true
    targets:
      - id: {target}
agents:
  - id: {caller}
    workspace: sample
    allow: ["{target}.hello"]
    deny:
      - capability: "{target}.goodbye"
        reason: sample allows hello only
""", encoding="utf-8")
            config = work / ".claude.json"
            original = {"mcpServers": {"sample": {"command": sys.executable, "args": [
                str(Path(__file__).resolve()), "--sample-server",
            ]}}}
            config.write_text(json.dumps(original), encoding="utf-8")
            cli("wrap", "add", "--wrap-bin", str(work / "harbor-wrap"), "--policy", str(policy),
                "--agent-id", caller, "--target-id", target, "claude-code", "sample")
            installed = json.loads(config.read_text())["mcpServers"]["sample"]

            def exchange(msg_id, method, params=None):
                return json.dumps({"jsonrpc": "2.0", "id": msg_id, "method": method,
                                   "params": params or {}})

            protocol = "2025-06-18"
            calls = [
                exchange(1, "initialize", {"protocolVersion": protocol, "capabilities": {},
                                           "clientInfo": {"name": "demo", "version": "1"}}),
                json.dumps({"jsonrpc": "2.0", "method": "notifications/initialized"}),
                exchange(2, "tools/list"),
                exchange(3, "tools/call", {"name": "hello"}),
                exchange(4, "tools/call", {"name": "goodbye"}),
            ]
            completed = subprocess.run([installed["command"], *installed["args"]], cwd=work,
                                       env=env, input="\n".join(calls) + "\n", text=True,
                                       capture_output=True, timeout=20, check=True)
            responses = {r["id"]: r for r in map(json.loads, completed.stdout.splitlines())}
            require(responses[1]["result"]["protocolVersion"] == protocol, "MCP did not initialize")
            require(len(responses[2]["result"]["tools"]) == 2, "sample tools missing")
            require(responses[3]["result"]["content"][0]["text"] == "hello", "hello did not run")
            denial = responses[4]["error"]["data"]
            require(denial["reason"] == "sample allows hello only", "policy denial missing")
            require("synced 2 events" in cli("sync"), "first sync did not send both records")
            require("synced 0 events" in cli("sync"), "checkpoint did not suppress repeat upload")
            require("synced 2 events" in cli("sync", "--replay"), "replay did not send both records")

            # Use the scoped API to check duplicates; the display CLI itself deduplicates.
            records = api("/api/v1/audit/traces?limit=20")["data"]
            require(len(records) == 2, "audit must contain exactly two records after replay")
            by_tool = {r["toolName"]: r for r in records}
            require(by_tool["hello"]["outcome"] == "ok", "successful call missing")
            require(by_tool["goodbye"]["id"] == denial["auditId"], "denial and audit IDs differ")
            require(all(r["callerAgentId"] == caller and r["targetAgentId"] == target
                        for r in records), "wrong registered audit identity")

            why = cli("why", denial["auditId"])
            require(all(x in why for x in (denial["auditId"], denial["reason"], denial["policy"])),
                    "harbor why did not explain the same denial")
            cli("wrap", "remove", "claude-code", "sample")
            require(json.loads(config.read_text()) == original, "uninstall did not restore config")
            print(why.rstrip())
            print(f"Verified: allowed + denied calls, 2 audit records, safe replay, restored config "
                  f"({time.monotonic() - started:.1f}s, including builds).")
        finally:
            if server.poll() is None:
                server.terminate()
                try:
                    server.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    server.kill()


if __name__ == "__main__":
    if sys.argv[1:] == ["--sample-server"]:
        sample_server()
    elif not sys.argv[1:]:
        with tempfile.TemporaryDirectory(prefix="harbor-stdio-") as temp:
            run_demo(Path(__file__).resolve().parents[1], Path(temp))
    else:
        sys.exit("usage: python3 scripts/stdio-audit-quickstart.py")
