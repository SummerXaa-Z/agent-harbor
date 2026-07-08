import { useEffect, useMemo, useState } from "react";

import { fetchAccessDecisionExplanation } from "../api";
import {
  buildExplainRequest,
  buildPermissionChangeHandoff,
  decisionRecordRows,
  type AskAccessSelection
} from "../askJourney";
import type { Translator } from "../consolePresenters";
import type { Language } from "../i18n";
import {
  localizedErrorMessageState,
  localizedMessageText,
  tx,
  type LocalizedMessage
} from "../localizedMessages";
import type {
  AccessDecisionExplainRequest,
  AccessDecisionExplainResult,
  AskHandoffContext,
  ConsoleData,
  PermissionChangeHandoffContext
} from "../types";
import type { PermissionPackageTemplate } from "../permissionPackages";

export interface AskAccessHistoryEntry {
  id: string
  request: AccessDecisionExplainRequest
  result: AccessDecisionExplainResult
}

interface UseAskAccessControllerArgs {
  adminKey: string
  consoleData: ConsoleData | null
  handoffContext: AskHandoffContext | null
  language: Language
  liveDataAvailable: boolean
  onConsumeHandoff: () => void
  onStartPermissionChange: (context: PermissionChangeHandoffContext) => void
  t: Translator
  templates: PermissionPackageTemplate[]
}

export function useAskAccessController({
  adminKey,
  consoleData,
  handoffContext,
  language,
  liveDataAvailable,
  onConsumeHandoff,
  onStartPermissionChange,
  t,
  templates
}: UseAskAccessControllerArgs) {
  const [selection, setSelection] = useState<AskAccessSelection>({});
  const [result, setResult] = useState<AccessDecisionExplainResult | null>(null);
  const [history, setHistory] = useState<AskAccessHistoryEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [messageState, setMessage] = useState<LocalizedMessage | null>(null);
  const message = localizedMessageText(messageState, t, language);

  const effectiveSelection = useMemo(
    () => applySelectionDefaults(selection, consoleData),
    [consoleData, selection]
  );
  const requestBuild = useMemo(() => buildExplainRequest(effectiveSelection), [effectiveSelection]);
  const chainRows = useMemo(() => result ? decisionRecordRows(result) : [], [result]);
  const exampleSelection = useMemo(() => applySelectionDefaults({}, consoleData), [consoleData]);
  const exampleAvailable = buildExplainRequest(exampleSelection).complete;

  useEffect(() => {
    if (!handoffContext) return;
    setSelection((current) => ({
      ...current,
      callerInstanceId: handoffContext.callerInstanceId ?? current.callerInstanceId,
      capabilityId: handoffContext.capabilityId ?? current.capabilityId,
      subjectId: handoffContext.subjectId ?? current.subjectId,
      targetId: handoffContext.targetId ?? current.targetId,
      tenantId: handoffContext.tenantId ?? current.tenantId,
      workspaceId: handoffContext.workspaceId ?? current.workspaceId
    }));
    setResult(null);
    setMessage(null);
    onConsumeHandoff();
  }, [handoffContext]);

  function updateSelection(next: AskAccessSelection) {
    setSelection((current) => {
      const merged = { ...current, ...next };
      if (next.targetId && next.targetId !== current.targetId) {
        return { ...merged, capabilityId: "" };
      }
      return merged;
    });
    setResult(null);
    setMessage(null);
  }

  async function explain(selectionOverride?: AskAccessSelection) {
    if (!consoleData) return;
    if (!liveDataAvailable) {
      setMessage({ key: "message.accessDecisionExplainRequiresLiveApi" });
      return;
    }
    const nextSelection = selectionOverride ?? effectiveSelection;
    const build = buildExplainRequest(nextSelection);
    if (!build.complete || !build.request) {
      setMessage({ key: "message.accessDecisionExplainMissingFields" });
      return;
    }
    const request = build.request;
    setLoading(true);
    setMessage(null);
    try {
      const next = await fetchAccessDecisionExplanation(request, adminKey);
      setResult(next);
      setHistory((current) => [
        {
          id: historyId(request),
          request,
          result: next
        },
        ...current.filter((entry) => historyId(entry.request) !== historyId(request))
      ].slice(0, 5));
      setMessage({ key: "message.accessDecisionExplainLoaded" });
    } catch (error) {
      setMessage(localizedErrorMessageState(error, "error.explainAccessDecision"));
    } finally {
      setLoading(false);
    }
  }

  function selectHistory(entry: AskAccessHistoryEntry) {
    setSelection(entry.request);
    setResult(entry.result);
    setMessage(null);
  }

  async function runExampleQuery() {
    setSelection(exampleSelection);
    await explain(exampleSelection);
  }

  function startPermissionChange() {
    if (!consoleData) return;
    const request = result?.request ?? requestBuild.request;
    if (!request) {
      setMessage({ key: "message.accessDecisionExplainMissingFields" });
      return;
    }
    onStartPermissionChange(buildPermissionChangeHandoff(request, consoleData, {
      templates,
      translateIntent: (key, values) => tx(t, key, values)
    }));
  }

  return {
    chainRows,
    effectiveSelection,
    exampleAvailable,
    explain,
    history,
    loading,
    message,
    requestBuild,
    result,
    runExampleQuery,
    selectHistory,
    selection,
    setMessage,
    startPermissionChange,
    updateSelection
  };
}

export type AskAccessController = ReturnType<typeof useAskAccessController>;

function applySelectionDefaults(selection: AskAccessSelection, consoleData: ConsoleData | null): AskAccessSelection {
  if (!consoleData) return selection;
  const targetIdsWithCapabilities = new Set(consoleData.capabilities.map((capability) => capability.targetId));
  const target = findAgent(consoleData, selection.targetId)
    ?? consoleData.agents.find((agent) => targetIdsWithCapabilities.has(agent.id) && agent.status === "active")
    ?? consoleData.agents.find((agent) => targetIdsWithCapabilities.has(agent.id));
  const caller = findAgent(consoleData, selection.callerInstanceId)
    ?? consoleData.agents.find((agent) => agent.status === "active" && agent.channelType === "local")
    ?? consoleData.agents.find((agent) => agent.status === "active");
  const tenant = consoleData.tenants.find((item) => item.id === selection.tenantId)
    ?? consoleData.tenants.find((item) => item.id === caller?.tenantId)
    ?? consoleData.tenants[0];
  const targetCapabilities = consoleData.capabilities.filter((capability) => capability.targetId === target?.id);
  const capability = targetCapabilities.find((item) => item.id === selection.capabilityId) ?? targetCapabilities[0];

  return {
    capabilityId: selection.capabilityId || capability?.id || "",
    callerInstanceId: selection.callerInstanceId || caller?.id || "",
    subjectId: selection.subjectId || "",
    targetId: selection.targetId || target?.id || "",
    tenantId: selection.tenantId || tenant?.id || caller?.tenantId || "",
    workspaceId: selection.workspaceId || caller?.workspaceId || target?.workspaceId || ""
  };
}

function findAgent(consoleData: ConsoleData, agentId?: string) {
  return agentId ? consoleData.agents.find((agent) => agent.id === agentId) : undefined;
}

function historyId(request: AccessDecisionExplainRequest) {
  return [
    request.tenantId,
    request.workspaceId,
    request.callerInstanceId,
    request.targetId,
    request.capabilityId,
    request.subjectId ?? ""
  ].join("|");
}
