import { accessNextActionKeys } from "../../askJourney.ts";
import type { AccessDecisionExplainEvidence, AccessDecisionExplainResult } from "../../types.ts";

// The gateway evaluates these layers in this order; explain omits the layers
// after the one that denied.
export const decisionChainLayers = [
  "caller_instance",
  "target",
  "capability",
  "tenant_entitlement",
  "workspace_assignment",
  "instance_assignment",
] as const;

export type DecisionChainLayer = (typeof decisionChainLayers)[number];
export type DecisionNodeState = "passed" | "failed" | "pending";

export interface DecisionChainNode {
  evidence?: AccessDecisionExplainEvidence;
  labelKey: string;
  layer: DecisionChainLayer;
  state: DecisionNodeState;
}

export interface DecisionChain {
  nodes: DecisionChainNode[];
  outcome: AccessDecisionExplainResult["outcome"];
  passedCount: number;
  terminatingLayer: DecisionChainLayer | null;
}

const failingStatuses = new Set(["blocking", "denied", "missing", "mismatch", "not_approved", "inactive", "pending_review"]);

export function buildDecisionChain(result: AccessDecisionExplainResult): DecisionChain {
  const evidenceByLayer = new Map<string, AccessDecisionExplainEvidence>();
  for (const record of result.evidence) {
    if (!evidenceByLayer.has(record.layer)) evidenceByLayer.set(record.layer, record);
  }

  if (result.outcome === "allowed") {
    const nodes = decisionChainLayers.map((layer) => node(layer, "passed", evidenceByLayer.get(layer)));
    return { nodes, outcome: result.outcome, passedCount: nodes.length, terminatingLayer: null };
  }

  const terminatingIndex = terminatingLayerIndex(result, evidenceByLayer);
  const nodes = decisionChainLayers.map((layer, index) => {
    const evidence = evidenceByLayer.get(layer);
    if (index === terminatingIndex) return node(layer, "failed", evidence);
    if (index > terminatingIndex) return node(layer, "pending", evidence);
    return node(layer, evidence?.status === "matched" ? "passed" : "pending", evidence);
  });
  return {
    nodes,
    outcome: result.outcome,
    passedCount: nodes.filter((item) => item.state === "passed").length,
    terminatingLayer: decisionChainLayers[terminatingIndex],
  };
}

// First failing record wins; without one, fall back to the layer the decision
// names as its source, then to the first layer explain left out.
function terminatingLayerIndex(result: AccessDecisionExplainResult, evidenceByLayer: Map<string, AccessDecisionExplainEvidence>) {
  const failing = decisionChainLayers.findIndex((layer) => {
    const status = evidenceByLayer.get(layer)?.status;
    return status !== undefined && failingStatuses.has(status);
  });
  if (failing !== -1) return failing;
  const source = decisionChainLayers.indexOf(result.decision.source as DecisionChainLayer);
  if (source !== -1) return source;
  const missing = decisionChainLayers.findIndex((layer) => !evidenceByLayer.has(layer));
  return missing === -1 ? decisionChainLayers.length - 1 : missing;
}

function node(layer: DecisionChainLayer, state: DecisionNodeState, evidence?: AccessDecisionExplainEvidence): DecisionChainNode {
  return { evidence, labelKey: `ask.recordLayer.${layer}`, layer, state };
}

export interface DecisionRemediation {
  code: string;
  fallback: string;
  labelKey: string | null;
  // Codes a permission package can fix, so the page offers "request access".
  requestable: boolean;
}

const requestableCodes = new Set(["use_permission_package", "create_workspace_assignment", "create_caller_assignment"]);

export function decisionRemediation(result: AccessDecisionExplainResult): DecisionRemediation {
  const code = result.nextActionCodes?.[0] ?? (result.outcome === "allowed" ? "no_change_required" : "inspect_access_profile");
  return {
    code,
    fallback: result.nextActions[0] ?? "",
    labelKey: accessNextActionKeys[code] ?? null,
    requestable: result.outcome === "denied" && requestableCodes.has(code),
  };
}
