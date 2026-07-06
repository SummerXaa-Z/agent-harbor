import { navHashFor, type NavKey } from "./consoleNavigation.ts";
import type { ConsoleData } from "./types";

export type ProductionJourneyStageKey =
  | "setup"
  | "resources"
  | "access_query"
  | "permission_change"
  | "go_live_status"
  | "handoff";

export type ProductionJourneyState =
  | "empty"
  | "partial"
  | "configured"
  | "denied"
  | "in_change"
  | "ready"
  | "blocked";

export interface ProductionJourneyStage {
  key: ProductionJourneyStageKey;
  labelKey: string;
}

export interface ProductionJourneyInput {
  accessOutcome?: "allowed" | "denied" | null;
  activeNav?: NavKey;
  data: ConsoleData;
  hasPermissionChangeContext?: boolean;
  permissionBlocked?: boolean;
  permissionReady?: boolean;
}

export interface ProductionJourney {
  completedStageKeys: ProductionJourneyStageKey[];
  currentStageKey: ProductionJourneyStageKey;
  nextActionHash: string;
  nextActionKey: string;
  state: ProductionJourneyState;
}

export const productionJourneyStages: ProductionJourneyStage[] = [
  { key: "setup", labelKey: "productionJourney.stage.setup" },
  { key: "resources", labelKey: "productionJourney.stage.resources" },
  { key: "access_query", labelKey: "productionJourney.stage.accessQuery" },
  { key: "permission_change", labelKey: "productionJourney.stage.permissionChange" },
  { key: "go_live_status", labelKey: "productionJourney.stage.goLiveStatus" },
  { key: "handoff", labelKey: "productionJourney.stage.handoff" }
];

export function deriveProductionJourney(input: ProductionJourneyInput): ProductionJourney {
  const setupComplete = isProductionSetupComplete(input.data);
  const hasLiveSetupData = input.data.setupLoadedFromApi;
  const hasAnyConfiguredResource =
    input.data.tenants.length > 0 ||
    input.data.agents.length > 0 ||
    input.data.capabilities.length > 0 ||
    input.data.tenantEntitlements.length > 0;

  if (!setupComplete) {
    return {
      completedStageKeys: [],
      currentStageKey: "setup",
      nextActionHash: hasAnyConfiguredResource ? "#registry" : "#getting-started",
      nextActionKey: hasAnyConfiguredResource
        ? "productionJourney.next.continueSetup"
        : "productionJourney.next.setup",
      state: hasLiveSetupData && hasAnyConfiguredResource ? "partial" : "empty"
    };
  }

  if (input.permissionBlocked) {
    return {
      completedStageKeys: ["setup", "resources", "access_query"],
      currentStageKey: "permission_change",
      nextActionHash: "#ai-admin",
      nextActionKey: "productionJourney.next.resolveBlocker",
      state: "blocked"
    };
  }

  if (input.permissionReady) {
    return {
      completedStageKeys: ["setup", "resources", "access_query", "permission_change"],
      currentStageKey: "go_live_status",
      nextActionHash: navHashFor("evidence"),
      nextActionKey: "productionJourney.next.confirmGoLive",
      state: "ready"
    };
  }

  if (input.hasPermissionChangeContext || input.activeNav === "ai-admin") {
    return {
      completedStageKeys: ["setup", "resources", "access_query"],
      currentStageKey: "permission_change",
      nextActionHash: "#ai-admin",
      nextActionKey: "productionJourney.next.completePermissionChange",
      state: "in_change"
    };
  }

  if (input.accessOutcome === "denied") {
    return {
      completedStageKeys: ["setup", "resources"],
      currentStageKey: "access_query",
      nextActionHash: "#ai-admin",
      nextActionKey: "productionJourney.next.fixDenied",
      state: "denied"
    };
  }

  return {
    completedStageKeys: ["setup", "resources"],
    currentStageKey: "access_query",
    nextActionHash: "#ask",
    nextActionKey: "productionJourney.next.ask",
    state: "configured"
  };
}

function isProductionSetupComplete(data: ConsoleData) {
  return (
    data.setupLoadedFromApi &&
    data.tenants.length > 0 &&
    data.agents.some((agent) => agent.status === "active") &&
    data.capabilities.length > 0 &&
    data.tenantEntitlements.length > 0
  );
}
