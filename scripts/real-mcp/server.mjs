import express from "express";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StreamableHTTPServerTransport } from "@modelcontextprotocol/sdk/server/streamableHttp.js";
import { z } from "zod";

const host = process.env.REAL_MCP_HOST || "127.0.0.1";
const port = Number.parseInt(process.env.REAL_MCP_PORT || "8787", 10);

const tools = [
  "search_customer",
  "update_ticket",
  "export_contracts"
];

function buildServer() {
  const server = new McpServer({
    name: "agent-harbor-real-mcp-demo",
    version: "0.1.0"
  });

  server.registerTool(
    "search_customer",
    {
      title: "Search Customer",
      description: "Search customer records from a live MCP SDK demo service.",
      inputSchema: {
        query: z.string().min(1).describe("Customer name, email, or ticket keyword to search for.")
      }
    },
    async ({ query }) => ({
      content: [
        {
          type: "text",
          text: JSON.stringify({
            source: "real-mcp-sdk",
            query,
            matches: [
              {
                customerId: "cust-001",
                name: "Acme Support",
                tier: "enterprise",
                region: "us-east"
              }
            ]
          })
        }
      ]
    })
  );

  server.registerTool(
    "update_ticket",
    {
      title: "Update Ticket",
      description: "Update support-ticket state from a live MCP SDK demo service.",
      inputSchema: {
        ticketId: z.string().min(1).describe("Support ticket identifier, for example TCK-1001."),
        status: z.enum(["open", "pending", "triaged", "resolved"]).describe("New ticket status.")
      }
    },
    async ({ ticketId, status }) => ({
      content: [
        {
          type: "text",
          text: JSON.stringify({
            source: "real-mcp-sdk",
            ticketId,
            status,
            updated: true
          })
        }
      ]
    })
  );

  server.registerTool(
    "export_contracts",
    {
      title: "Export Contracts",
      description: "Export customer contracts from a live MCP SDK demo service. This is intentionally high-risk.",
      inputSchema: {
        format: z.enum(["csv", "json"]).describe("Export format.")
      }
    },
    async ({ format }) => ({
      content: [
        {
          type: "text",
          text: JSON.stringify({
            source: "real-mcp-sdk",
            format,
            exportId: "contract-export-demo",
            status: "prepared"
          })
        }
      ]
    })
  );

  return server;
}

const app = express();
app.use((req, res, next) => {
  res.setHeader("Access-Control-Allow-Origin", process.env.REAL_MCP_CORS_ORIGIN || "*");
  res.setHeader("Access-Control-Allow-Headers", "Accept, Content-Type");
  res.setHeader("Access-Control-Allow-Methods", "GET, POST, OPTIONS");
  if (req.method === "OPTIONS") {
    res.sendStatus(204);
    return;
  }
  next();
});
app.use(express.json({ limit: "1mb" }));

app.get("/", (_req, res) => {
  res.json({ status: "ok", server: "agent-harbor-real-mcp-demo", tools });
});

app.get("/healthz", (_req, res) => {
  res.json({ status: "ok", server: "agent-harbor-real-mcp-demo", tools });
});

app.post("/mcp", async (req, res) => {
  const server = buildServer();
  const transport = new StreamableHTTPServerTransport({
    sessionIdGenerator: undefined,
    enableJsonResponse: true
  });

  res.on("close", () => {
    transport.close();
    server.close();
  });

  try {
    await server.connect(transport);
    await transport.handleRequest(req, res, req.body);
  } catch (error) {
    console.error("real MCP request failed", error);
    if (!res.headersSent) {
      res.status(500).json({
        jsonrpc: "2.0",
        id: req.body?.id ?? null,
        error: {
          code: -32603,
          message: "real MCP demo request failed"
        }
      });
    }
  }
});

app.listen(port, host, () => {
  console.log(`real MCP SDK server listening on http://${host}:${port}/mcp`);
});
