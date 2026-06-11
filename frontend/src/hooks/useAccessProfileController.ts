import { useEffect, useState, type Dispatch, type SetStateAction } from "react";
import { fetchAccessDecisionExplanation, loadTenantAccessProfile } from "../api";
import { parseAccessProfileTraceLimit } from "../accessProfile";
import type { Translator } from "../consolePresenters";
import type { NavKey } from "../consoleNavigation";
import type { Language } from "../i18n";
import type {
  AccessDecisionExplainRequest,
  AccessDecisionExplainResult,
  AccessProfileFilters,
  AccessProfileHandoffContext,
  ManagementScope,
  TenantAccessProfileData
} from "../types";

const defaultAccessProfileFilters: AccessProfileFilters = {
  callerInstanceId: "",
  capabilityId: "",
  subjectId: "",
  targetId: "",
  traceLimit: "20",
  workspaceId: ""
};

interface UseAccessProfileControllerArgs {
  activeNav: NavKey;
  adminKey: string;
  dataLoadedFromApi: boolean;
  defaultScope: ManagementScope;
  language: Language;
  scope: ManagementScope;
  setScope: Dispatch<SetStateAction<ManagementScope>>;
  t: Translator;
}

export function useAccessProfileController({
  activeNav,
  adminKey,
  dataLoadedFromApi,
  defaultScope,
  language,
  scope,
  setScope,
  t
}: UseAccessProfileControllerArgs) {
  const [filters, setFilters] = useState<AccessProfileFilters>(defaultAccessProfileFilters);
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState("");
  const [profile, setProfile] = useState<TenantAccessProfileData | null>(null);
  const [handoffContext, setHandoffContext] = useState<AccessProfileHandoffContext | null>(null);
  const [decisionExplanation, setDecisionExplanation] = useState<AccessDecisionExplainResult | null>(null);
  const [decisionExplainLoading, setDecisionExplainLoading] = useState(false);
  const [decisionExplainMessage, setDecisionExplainMessage] = useState("");

  useEffect(() => {
    if (activeNav === "access" && !profile && !loading) {
      void refresh();
    }
  }, [activeNav]);

  function updateFilters(nextFilters: AccessProfileFilters) {
    setFilters(nextFilters);
    setHandoffContext(null);
    setDecisionExplanation(null);
    setDecisionExplainMessage("");
  }

  function changeTenant(tenantId: string) {
    setScope((current) => ({ ...current, tenantId }));
    setHandoffContext(null);
    setProfile(null);
    setDecisionExplanation(null);
    setDecisionExplainMessage("");
  }

  function clearForPermissionChangeHandoff(nextFilters: AccessProfileFilters, nextContext: AccessProfileHandoffContext) {
    setFilters(nextFilters);
    setHandoffContext(nextContext);
    setProfile(null);
    setMessage("");
    setDecisionExplanation(null);
    setDecisionExplainMessage("");
  }

  async function refresh() {
    const traceLimit = parseAccessProfileTraceLimit(filters.traceLimit);
    if (!traceLimit.ok) {
      setMessage(traceLimit.message);
      return;
    }
    const requestScope = normalizedScope(scope, defaultScope);
    setLoading(true);
    setMessage("");
    try {
      const next = await loadTenantAccessProfile(requestScope.tenantId, adminKey, {
        ...filters,
        traceLimit: traceLimit.value
      });
      setProfile(next);
      setMessage(next.loadedFromApi ? t("status.profileRefreshed") : t("status.profileFallback"));
    } catch (error) {
      setMessage(localizedErrorMessage(t, language, error, "error.loadTenantAccessProfile"));
    } finally {
      setLoading(false);
    }
  }

  async function explainAccessDecision() {
    if (!dataLoadedFromApi) {
      setDecisionExplainMessage(t("message.accessDecisionExplainRequiresLiveApi"));
      return;
    }
    const requestScope = normalizedScope(scope, defaultScope);
    const request: AccessDecisionExplainRequest = {
      callerInstanceId: filters.callerInstanceId?.trim() ?? "",
      capabilityId: filters.capabilityId?.trim() ?? "",
      subjectId: filters.subjectId?.trim() || undefined,
      targetId: filters.targetId?.trim() ?? "",
      tenantId: requestScope.tenantId,
      workspaceId: filters.workspaceId?.trim() || requestScope.workspaceId
    };
    if (!accessDecisionExplainRequestComplete(request)) {
      setDecisionExplainMessage(t("message.accessDecisionExplainMissingFields"));
      return;
    }
    setDecisionExplainLoading(true);
    setDecisionExplainMessage("");
    try {
      const next = await fetchAccessDecisionExplanation(request, adminKey);
      setDecisionExplanation(next);
      setDecisionExplainMessage(t("message.accessDecisionExplainLoaded"));
    } catch (error) {
      setDecisionExplainMessage(localizedErrorMessage(t, language, error, "error.explainAccessDecision"));
    } finally {
      setDecisionExplainLoading(false);
    }
  }

  return {
    changeTenant,
    clearForPermissionChangeHandoff,
    decisionExplainLoading,
    decisionExplainMessage,
    decisionExplanation,
    explainAccessDecision,
    filters,
    handoffContext,
    loading,
    message,
    profile,
    refresh,
    setDecisionExplainMessage,
    setDecisionExplanation,
    setFilters,
    setHandoffContext,
    setMessage,
    setProfile,
    updateFilters
  };
}

function localizedErrorMessage(t: Translator, language: Language, error: unknown, fallbackKey: string) {
  const fallback = t(fallbackKey);
  if (!(error instanceof Error) || !error.message.trim()) return fallback;
  if (language === "en" || /[\u4e00-\u9fa5]/.test(error.message)) {
    return error.message;
  }
  return fallback;
}

function normalizedScope(scope: ManagementScope, defaultScope: ManagementScope): ManagementScope {
  return {
    tenantId: scope.tenantId.trim() || defaultScope.tenantId,
    workspaceId: scope.workspaceId.trim() || defaultScope.workspaceId
  };
}

function accessDecisionExplainRequestComplete(request: AccessDecisionExplainRequest) {
  return Boolean(
    request.callerInstanceId.trim() &&
    request.capabilityId.trim() &&
    request.targetId.trim() &&
    request.tenantId.trim() &&
    request.workspaceId.trim()
  );
}
