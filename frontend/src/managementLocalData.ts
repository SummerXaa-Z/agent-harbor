import type { Agent, ConsoleData, RoutePolicy } from "./types";

export function mergeManagementAgentIntoConsoleData(current: ConsoleData, agent: Agent): ConsoleData {
  return {
    ...current,
    agents: upsertById(current.agents, agent)
  };
}

export function mergeManagementRoutePolicyIntoConsoleData(current: ConsoleData, policy: RoutePolicy): ConsoleData {
  return {
    ...current,
    routePolicies: upsertById(current.routePolicies, policy),
    routePoliciesLoadedFromApi: true
  };
}

function upsertById<T extends { id: string }>(rows: T[], next: T) {
  const found = rows.some((row) => row.id === next.id);
  if (!found) return [next, ...rows];
  return rows.map((row) => (row.id === next.id ? next : row));
}
