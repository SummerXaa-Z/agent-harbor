import type { Translator } from "./consolePresenters";
import type { CoreJourneyStep } from "./coreJourney";

type CoreJourneyStepDetailStatus = "complete" | "partial" | "missing";

const stepDetailKeys: Record<CoreJourneyStep["key"], Record<CoreJourneyStepDetailStatus, string>> = {
  accessProfile: {
    complete: "coreJourney.stepDetail.accessProfile.complete",
    missing: "coreJourney.stepDetail.accessProfile.missing",
    partial: "coreJourney.stepDetail.accessProfile.partial"
  },
  agentPair: {
    complete: "coreJourney.stepDetail.agentPair.complete",
    missing: "coreJourney.stepDetail.agentPair.missing",
    partial: "coreJourney.stepDetail.agentPair.partial"
  },
  capabilityDiscovery: {
    complete: "coreJourney.stepDetail.capabilityDiscovery.complete",
    missing: "coreJourney.stepDetail.capabilityDiscovery.missing",
    partial: "coreJourney.stepDetail.capabilityDiscovery.partial"
  },
  grantChain: {
    complete: "coreJourney.stepDetail.grantChain.complete",
    missing: "coreJourney.stepDetail.grantChain.missing",
    partial: "coreJourney.stepDetail.grantChain.partial"
  },
  runtimeEvidence: {
    complete: "coreJourney.stepDetail.runtimeEvidence.complete",
    missing: "coreJourney.stepDetail.runtimeEvidence.missing",
    partial: "coreJourney.stepDetail.runtimeEvidence.partial"
  },
  tenantTree: {
    complete: "coreJourney.stepDetail.tenantTree.complete",
    missing: "coreJourney.stepDetail.tenantTree.missing",
    partial: "coreJourney.stepDetail.tenantTree.partial"
  }
};

export function coreJourneyStepDetailLabel(step: CoreJourneyStep, t: Translator): string {
  return t(stepDetailKeys[step.key][step.status]);
}
