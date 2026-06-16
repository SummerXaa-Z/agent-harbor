import { useEffect, useState, type Dispatch, type SetStateAction } from "react";
import { loadTenantAccessProfile } from "../api";
import { parseAccessProfileTraceLimit } from "../accessProfile";
import type { Translator } from "../consolePresenters";
import type { NavKey } from "../consoleNavigation";
import type { Language } from "../i18n";
import {
  localizedErrorMessageState,
  localizedMessageText,
  type LocalizedMessage
} from "../localizedMessages";
import type {
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
  defaultScope: ManagementScope;
  enabled: boolean;
  language: Language;
  scope: ManagementScope;
  setScope: Dispatch<SetStateAction<ManagementScope>>;
  t: Translator;
}

export function useAccessProfileController({
  activeNav,
  adminKey,
  defaultScope,
  enabled,
  language,
  scope,
  setScope,
  t
}: UseAccessProfileControllerArgs) {
  const [filters, setFilters] = useState<AccessProfileFilters>(defaultAccessProfileFilters);
  const [loading, setLoading] = useState(false);
  const [messageState, setMessage] = useState<LocalizedMessage | null>(null);
  const [profile, setProfile] = useState<TenantAccessProfileData | null>(null);
  const [handoffContext, setHandoffContext] = useState<AccessProfileHandoffContext | null>(null);
  const message = localizedMessageText(messageState, t, language);

  useEffect(() => {
    if (enabled && activeNav === "access" && !profile && !loading) {
      void refresh();
    }
  }, [activeNav, enabled]);

  function updateFilters(nextFilters: AccessProfileFilters) {
    setFilters(nextFilters);
    setHandoffContext(null);
  }

  function changeTenant(tenantId: string) {
    setScope((current) => ({ ...current, tenantId }));
    setHandoffContext(null);
    setProfile(null);
  }

  function clearForPermissionChangeHandoff(nextFilters: AccessProfileFilters, nextContext: AccessProfileHandoffContext) {
    setFilters(nextFilters);
    setHandoffContext(nextContext);
    setProfile(null);
    setMessage(null);
  }

  async function refresh() {
    if (!enabled) return;
    const traceLimit = parseAccessProfileTraceLimit(filters.traceLimit);
    if (!traceLimit.ok) {
      setMessage({ key: "message.validationTraceLimit" });
      return;
    }
    const requestScope = normalizedScope(scope, defaultScope);
    setLoading(true);
    setMessage(null);
    try {
      const next = await loadTenantAccessProfile(requestScope.tenantId, adminKey, {
        ...filters,
        traceLimit: traceLimit.value
      });
      setProfile(next);
      setMessage({ key: next.loadedFromApi ? "status.profileRefreshed" : "status.profileFallback" });
    } catch (error) {
      setMessage(localizedErrorMessageState(error, "error.loadTenantAccessProfile"));
    } finally {
      setLoading(false);
    }
  }

  return {
    changeTenant,
    clearForPermissionChangeHandoff,
    filters,
    handoffContext,
    loading,
    message,
    profile,
    refresh,
    setFilters,
    setHandoffContext,
    setMessage,
    setProfile,
    updateFilters
  };
}

function normalizedScope(scope: ManagementScope, defaultScope: ManagementScope): ManagementScope {
  return {
    tenantId: scope.tenantId.trim() || defaultScope.tenantId,
    workspaceId: scope.workspaceId.trim() || defaultScope.workspaceId
  };
}
