import { useEffect, useMemo, useRef, useState } from "react";

import { fetchAccessDecisionExplanation } from "../api";
import {
  buildExplainRequest,
  buildPermissionChangeHandoff,
  canStartPermissionChangeForAdmin,
  decisionRecordRows,
  resolveAskAccessSelection,
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
  adminRole?: string
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
  adminRole,
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
  const activeRequestRef = useRef<AbortController | null>(null);
  const requestSequenceRef = useRef(0);
  const message = localizedMessageText(messageState, t, language);

  const effectiveSelection = useMemo(
    () => consoleData ? resolveAskAccessSelection(selection, consoleData) : selection,
    [consoleData, selection]
  );
  const requestBuild = useMemo(() => buildExplainRequest(effectiveSelection), [effectiveSelection]);
  const permissionChangeAvailable = useMemo(
    () => canStartPermissionChangeForAdmin(adminRole, result?.request ?? requestBuild.request ?? undefined, consoleData?.agents ?? []),
    [adminRole, consoleData, requestBuild.request, result]
  );
  const recordRows = useMemo(() => result ? decisionRecordRows(result) : [], [result]);
  const exampleSelection = useMemo(
    () => consoleData ? resolveAskAccessSelection({}, consoleData) : {},
    [consoleData]
  );
  const exampleAvailable = buildExplainRequest(exampleSelection).complete;

  useEffect(() => {
    if (!handoffContext || !consoleData) return;
    setSelection((current) => resolveAskAccessSelection({
      ...current,
      callerInstanceId: handoffContext.callerInstanceId ?? current.callerInstanceId,
      capabilityId: handoffContext.capabilityId ?? current.capabilityId,
      subjectId: handoffContext.subjectId ?? current.subjectId,
      targetId: handoffContext.targetId ?? current.targetId,
      tenantId: handoffContext.tenantId ?? current.tenantId,
      workspaceId: handoffContext.workspaceId ?? current.workspaceId
    }, consoleData));
    invalidateActiveRequest();
    setResult(null);
    setMessage(null);
    onConsumeHandoff();
  }, [handoffContext, consoleData]);

  useEffect(() => () => activeRequestRef.current?.abort(), []);

  function updateSelection(next: AskAccessSelection) {
    invalidateActiveRequest();
    setSelection((current) => consoleData
      ? resolveAskAccessSelection({ ...current, ...next }, consoleData)
      : { ...current, ...next });
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
    activeRequestRef.current?.abort();
    const controller = new AbortController();
    const requestSequence = ++requestSequenceRef.current;
    activeRequestRef.current = controller;
    setLoading(true);
    setResult(null);
    setMessage(null);
    try {
      const next = await fetchAccessDecisionExplanation(request, adminKey, controller.signal);
      if (requestSequence !== requestSequenceRef.current) return;
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
      if (requestSequence !== requestSequenceRef.current || isAbortError(error)) return;
      setMessage(localizedErrorMessageState(error, "error.explainAccessDecision"));
    } finally {
      if (requestSequence === requestSequenceRef.current) {
        activeRequestRef.current = null;
        setLoading(false);
      }
    }
  }

  function selectHistory(entry: AskAccessHistoryEntry) {
    const nextSelection = consoleData
      ? resolveAskAccessSelection(entry.request, consoleData)
      : entry.request;
    setSelection(nextSelection);
    setResult(null);
    setMessage(null);
    void explain(nextSelection);
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
    if (!canStartPermissionChangeForAdmin(adminRole, request, consoleData.agents)) {
      setMessage({ key: "message.permissionChangeTargetManagedByPlatform" });
      return;
    }
    onStartPermissionChange(buildPermissionChangeHandoff(request, consoleData, {
      decisionResult: result ?? undefined,
      templates,
      translateIntent: (key, values) => tx(t, key, values)
    }));
  }

  function invalidateActiveRequest() {
    requestSequenceRef.current += 1;
    activeRequestRef.current?.abort();
    activeRequestRef.current = null;
    setLoading(false);
  }

  return {
    effectiveSelection,
    exampleAvailable,
    explain,
    history,
    loading,
    message,
    permissionChangeAvailable,
    requestBuild,
    recordRows,
    result,
    runExampleQuery,
    selectHistory,
    selection,
    setMessage,
    startPermissionChange,
    templates,
    updateSelection
  };
}

export type AskAccessController = ReturnType<typeof useAskAccessController>;

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

function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === "AbortError";
}
