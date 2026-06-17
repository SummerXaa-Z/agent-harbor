import type { Translator } from "./consolePresenters";
import type { AgentCreateFormState } from "./components/ManagementForms";
import type { ResourceLifecycleSetupGapKind } from "./resourceLifecycle";

export function setupGapAgentDraft(kind: ResourceLifecycleSetupGapKind, t: Translator): AgentCreateFormState {
  return {
    channelType: kind === "target" ? "mcp" : "local",
    credentialHeader: "",
    credentialName: "",
    credentialValue: "",
    description: kind === "target"
      ? t("resource.setupGap.target.agentDescription")
      : t("resource.setupGap.caller.agentDescription"),
    endpoint: "",
    name: kind === "target" ? t("resource.setupGap.target.agentName") : t("resource.setupGap.caller.agentName"),
    retryBackoffMs: "0",
    retryMaxAttempts: "1",
    status: "active"
  };
}
