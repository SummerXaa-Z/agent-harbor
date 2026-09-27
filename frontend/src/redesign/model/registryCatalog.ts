import type { Agent } from "../../types";

// Resource registry view-model. Channel kinds collapse to the two rows the
// governance console reasons about: local callers and upstream targets.

export interface RegistryRow {
  agent: Agent;
  endpoint: string;
  kind: "caller" | "target";
}

export function registryRows(agents: readonly Agent[]): RegistryRow[] {
  return agents
    .map((agent): RegistryRow => ({
      agent,
      endpoint: agentEndpoint(agent),
      kind: agent.channelType === "local" ? "caller" : "target"
    }))
    .sort((a, b) => a.kind.localeCompare(b.kind) || a.agent.name.localeCompare(b.agent.name));
}

export function agentEndpoint(agent: Agent): string {
  const endpoint = (agent.channelConfig as { endpoint?: unknown } | null | undefined)?.endpoint;
  return typeof endpoint === "string" ? endpoint : "";
}

export interface RegistrySummary {
  callers: number;
  targets: number;
}

export function registrySummary(rows: readonly RegistryRow[]): RegistrySummary {
  return {
    callers: rows.filter((row) => row.kind === "caller").length,
    targets: rows.filter((row) => row.kind === "target").length
  };
}
