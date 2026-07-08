import type { ConsoleData } from "./types";
import { navHashFor } from "./consoleNavigation.ts";

export type GettingStartedStepKey =
  | "connect-api"
  | "register-agents"
  | "discover-capabilities"
  | "create-grant-chain"
  | "run-decision"
  | "confirm-status";

export interface GettingStartedStep {
  key: GettingStartedStepKey;
  done: boolean;
  targetHash: string;
}

export function gettingStartedSteps(data: ConsoleData): GettingStartedStep[] {
  const setupDataAvailable = data.setupLoadedFromApi

  return [
    {
      done: setupDataAvailable,
      key: "connect-api",
      targetHash: "#getting-started"
    },
    {
      done: setupDataAvailable && data.tenants.length > 0 && data.agents.some((agent) => agent.status === "active"),
      key: "register-agents",
      targetHash: "#tenants"
    },
    {
      done: setupDataAvailable && data.capabilities.length > 0,
      key: "discover-capabilities",
      targetHash: "#capabilities"
    },
    {
      done: setupDataAvailable && data.tenantEntitlements.length > 0,
      key: "create-grant-chain",
      targetHash: "#ai-admin"
    },
    {
      done: data.loadedFromApi && data.traces.length > 0,
      key: "run-decision",
      targetHash: "#traces"
    },
    {
      done: data.loadedFromApi && data.evidenceRuns.length > 0,
      key: "confirm-status",
      targetHash: navHashFor("go-live")
    }
  ];
}

export function isSetupComplete(data: ConsoleData) {
  return gettingStartedSteps(data).slice(0, 4).every((step) => step.done);
}

export function resolveDefaultNavKey(data: ConsoleData): "ask" | "getting-started" {
  return isSetupComplete(data) ? "ask" : "getting-started";
}
