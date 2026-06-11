import type { ConsoleData } from "./types";

export type GettingStartedStepKey =
  | "connect-api"
  | "register-agents"
  | "discover-capabilities"
  | "create-grant-chain"
  | "run-decision"
  | "review-evidence";

export interface GettingStartedStep {
  key: GettingStartedStepKey;
  done: boolean;
  targetHash: string;
}

export function gettingStartedSteps(data: ConsoleData): GettingStartedStep[] {
  return [
    {
      done: data.loadedFromApi,
      key: "connect-api",
      targetHash: "#getting-started"
    },
    {
      done: data.tenants.length > 0 && data.agents.some((agent) => agent.status === "active"),
      key: "register-agents",
      targetHash: "#registry"
    },
    {
      done: data.capabilities.length > 0,
      key: "discover-capabilities",
      targetHash: "#capabilities"
    },
    {
      done: data.tenantEntitlements.length > 0,
      key: "create-grant-chain",
      targetHash: "#ai-admin"
    },
    {
      done: data.traces.length > 0,
      key: "run-decision",
      targetHash: "#traces"
    },
    {
      done: data.evidenceRuns.length > 0,
      key: "review-evidence",
      targetHash: "#evidence"
    }
  ];
}

export function isSetupComplete(data: ConsoleData) {
  return gettingStartedSteps(data).slice(0, 4).every((step) => step.done);
}

export function resolveDefaultNavKey(data: ConsoleData): "ask" | "getting-started" {
  return isSetupComplete(data) ? "ask" : "getting-started";
}
