#!/usr/bin/env python3
import argparse
import json
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


TOOLS = [
    {
        "name": "search_customer",
        "title": "Search Customer",
        "description": "Search customer records inside the mock CRM dataset.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "query": {"type": "string"},
            },
        },
        "outputSchema": {
            "type": "object",
            "properties": {
                "result": {"type": "string"},
            },
        },
    },
    {
        "name": "export_contracts",
        "title": "Export Contracts",
        "description": "Export contract records from the mock contract dataset.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "format": {"type": "string"},
            },
        },
        "outputSchema": {
            "type": "object",
            "properties": {
                "status": {"type": "string"},
            },
        },
    },
    {
        "name": "update_ticket",
        "title": "Update Ticket",
        "description": "Update a support ticket inside the mock support dataset.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "ticketId": {"type": "string"},
                "status": {"type": "string"},
            },
        },
        "outputSchema": {
            "type": "object",
            "properties": {
                "status": {"type": "string"},
            },
        },
    },
]


class MockMCPHandler(BaseHTTPRequestHandler):
    def do_OPTIONS(self):
        self.send_response(204)
        self.write_common_headers()
        self.end_headers()

    def do_POST(self):
        length = int(self.headers.get("Content-Length", "0"))
        try:
            request = json.loads(self.rfile.read(length) or b"{}")
        except json.JSONDecodeError:
            self.write_json({"jsonrpc": "2.0", "id": None, "error": {"code": -32700, "message": "parse error"}}, 400)
            return

        method = request.get("method")
        request_id = request.get("id")
        if method == "tools/list":
            self.write_json({"jsonrpc": "2.0", "id": request_id, "result": {"tools": TOOLS}})
            return
        if method == "tools/call":
            tool_name = (request.get("params") or {}).get("name")
            if tool_name not in {tool["name"] for tool in TOOLS}:
                self.write_json(
                    {"jsonrpc": "2.0", "id": request_id, "error": {"code": -32602, "message": "unknown tool"}},
                    400,
                )
                return
            self.write_json(
                {
                    "jsonrpc": "2.0",
                    "id": request_id,
                    "result": {
                        "content": [
                            {
                                "type": "text",
                                "text": f"mock result from {tool_name}",
                            }
                        ],
                        "isError": False,
                    },
                }
            )
            return

        self.write_json({"jsonrpc": "2.0", "id": request_id, "error": {"code": -32601, "message": "method not found"}}, 404)

    def do_GET(self):
        if self.path in {"/", "/healthz"}:
            self.write_json({"status": "ok", "tools": [tool["name"] for tool in TOOLS]})
            return
        self.write_json({"error": "not found"}, 404)

    def write_json(self, payload, status=200):
        body = json.dumps(payload, separators=(",", ":")).encode("utf-8")
        self.send_response(status)
        self.write_common_headers()
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def write_common_headers(self):
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type")

    def log_message(self, format, *args):
        return


def main():
    parser = argparse.ArgumentParser(description="Run a local mock MCP JSON-RPC server for AgentHarbor scenarios.")
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", default=8787, type=int)
    args = parser.parse_args()

    server = ThreadingHTTPServer((args.host, args.port), MockMCPHandler)
    print(f"mock MCP server listening on http://{args.host}:{args.port}/mcp", flush=True)
    server.serve_forever()


if __name__ == "__main__":
    main()
