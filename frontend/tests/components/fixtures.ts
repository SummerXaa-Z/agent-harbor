import type { Capability } from "../../src/types";

export function capability(overrides: Partial<Capability> = {}): Capability {
  return {
    id: "cap_1",
    targetId: "tgt_1",
    type: "mcp_tool",
    key: "crm.lookupCustomer",
    displayName: "Lookup Customer",
    action: "read",
    sensitivity: "internal",
    riskLevel: "low",
    enforcementMode: "gateway",
    discoveryStatus: "approved",
    version: 1,
    discoveredAt: "2026-11-01T00:00:00Z",
    updatedAt: "2026-11-01T00:00:00Z",
    ...overrides,
  };
}
