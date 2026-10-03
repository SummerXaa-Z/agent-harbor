import { useEffect, useRef, useState } from "react";

import {
  createAccessHandoffToken,
  fetchAccessHandoff,
  reportAccessHandoffConfigEvent,
  revokeAccessHandoffToken
} from "../api";
import type { Translator } from "../consolePresenters";
import type { Language } from "../i18n";
import { localizedErrorMessageState, localizedMessageText, type LocalizedMessage } from "../localizedMessages";
import type {
  AccessHandoff,
  AccessHandoffConfigAction,
  CreateAccessHandoffTokenResponse,
  PermissionPackageProductionReadinessFilter
} from "../permissionPackages";

interface UseAccessHandoffControllerArgs {
  adminKey: string;
  enabled: boolean;
  filter: PermissionPackageProductionReadinessFilter;
  language: Language;
  refreshKey?: string;
  t: Translator;
}

const accessHandoffSilentRefreshMs = 20000;

export function useAccessHandoffController({
  adminKey,
  enabled,
  filter,
  language,
  refreshKey = "",
  t
}: UseAccessHandoffControllerArgs) {
  const [handoff, setHandoff] = useState<AccessHandoff | null>(null);
  const [loading, setLoading] = useState(false);
  const [messageState, setMessage] = useState<LocalizedMessage | null>(null);
  const [oneTimeToken, setOneTimeToken] = useState<CreateAccessHandoffTokenResponse | null>(null);
  const [tokenAction, setTokenAction] = useState<"" | "create" | "revoke">("");
  const tokenMutationRef = useRef<"" | "create" | "revoke">("");
  const filterKey = accessHandoffFilterKey(filter);
  // The effect below keys on filterKey so object identity never refetches;
  // it reads the live filter through this ref to keep deps honest.
  const filterRef = useRef(filter);
  useEffect(() => {
    filterRef.current = filter;
  });

  useEffect(() => {
    setOneTimeToken(null);
    if (!enabled || !accessHandoffFilterReady(filterRef.current)) {
      setHandoff(null);
      setMessage(null);
      return;
    }
    const controller = new AbortController();
    setLoading(true);
    setMessage(null);
    fetchAccessHandoff(filterRef.current, adminKey, controller.signal)
      .then(setHandoff)
      .catch((error) => {
        if (controller.signal.aborted) return;
        setHandoff(null);
        setMessage(localizedErrorMessageState(error, "error.loadAccessHandoff"));
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    // Tokens can be revoked or expire outside this console (API, CLI), so keep
    // the delivery list fresh while the panel is open. Silent: no loading or
    // message churn, and never concurrent with a create/revoke mutation.
    const silentLoad = () => {
      if (tokenMutationRef.current || document.visibilityState !== "visible") return;
      fetchAccessHandoff(filterRef.current, adminKey).then(setHandoff).catch(() => undefined);
    };
    const interval = window.setInterval(silentLoad, accessHandoffSilentRefreshMs);
    const onVisibilityChange = () => {
      if (document.visibilityState === "visible") silentLoad();
    };
    document.addEventListener("visibilitychange", onVisibilityChange);
    return () => {
      controller.abort();
      window.clearInterval(interval);
      document.removeEventListener("visibilitychange", onVisibilityChange);
    };
  }, [adminKey, enabled, filterKey, refreshKey]);

  async function refresh() {
    if (!accessHandoffFilterReady(filter)) return null;
    setLoading(true);
    setMessage(null);
    try {
      const next = await fetchAccessHandoff(filter, adminKey);
      setHandoff(next);
      return next;
    } catch (error) {
      setMessage(localizedErrorMessageState(error, "error.loadAccessHandoff"));
      return null;
    } finally {
      setLoading(false);
    }
  }

  async function createToken(expiresInSeconds?: number) {
    if (tokenMutationRef.current || !handoff?.id || !handoff.tokenEligibility.eligible) return null;
    tokenMutationRef.current = "create";
    setTokenAction("create");
    setOneTimeToken(null);
    setMessage(null);
    try {
      const created = await createAccessHandoffToken({
        ...filter,
        expiresInSeconds: expiresInSeconds ?? handoff.tokenEligibility.defaultExpiresInSeconds,
        handoffId: handoff.id
      }, adminKey);
      setOneTimeToken(created);
      try {
        const next = await fetchAccessHandoff(filter, adminKey);
        setHandoff(next);
        setMessage({ key: "message.accessHandoffTokenCreated" });
      } catch {
        setMessage({ key: "message.accessHandoffTokenCreatedReloadFailed" });
      }
      return created;
    } catch (error) {
      setMessage(localizedErrorMessageState(error, "error.createAccessHandoffToken"));
      return null;
    } finally {
      tokenMutationRef.current = "";
      setTokenAction("");
    }
  }

  // Config-interaction auditing (evaluation finding 6): previewing or copying
  // the handoff config is reported to the server so it lands in the audit
  // trail. Best-effort by design — a reporting failure must never block or
  // fail the user's copy action after the clipboard already changed.
  function reportConfigEvent(action: AccessHandoffConfigAction) {
    if (!handoff?.id || !handoff.copyArtifacts) return;
    void reportAccessHandoffConfigEvent({
      ...filter,
      action,
      handoffId: handoff.id
    }, adminKey).catch(() => undefined);
  }

  async function revokeToken(id: string) {
    if (tokenMutationRef.current || !id.trim()) return null;
    tokenMutationRef.current = "revoke";
    setTokenAction("revoke");
    setMessage(null);
    try {
      const revoked = await revokeAccessHandoffToken(id, adminKey);
      setHandoff((current) => current ? {
        ...current,
        tokens: current.tokens.map((token) => token.id === revoked.id ? revoked : token)
      } : current);
      if (oneTimeToken?.id === id) setOneTimeToken(null);
      try {
        const next = await fetchAccessHandoff(filter, adminKey);
        setHandoff(next);
        setMessage({ key: "message.accessHandoffTokenRevoked" });
      } catch {
        setMessage({ key: "message.accessHandoffTokenRevokedReloadFailed" });
      }
      return revoked;
    } catch (error) {
      setMessage(localizedErrorMessageState(error, "error.revokeAccessHandoffToken"));
      return null;
    } finally {
      tokenMutationRef.current = "";
      setTokenAction("");
    }
  }

  return {
    clearOneTimeToken: () => setOneTimeToken(null),
    createToken,
    handoff,
    loading,
    message: localizedMessageText(messageState, t, language),
    oneTimeToken,
    refresh,
    reportConfigEvent,
    revokeToken,
    tokenAction
  };
}

function accessHandoffFilterReady(filter: PermissionPackageProductionReadinessFilter) {
  return [filter.callerInstanceId, filter.targetId, filter.templateId, filter.tenantId, filter.workspaceId]
    .every((value) => value.trim());
}

function accessHandoffFilterKey(filter: PermissionPackageProductionReadinessFilter) {
  return JSON.stringify([
    filter.approvalRequestId ?? "",
    filter.callerInstanceId,
    filter.region ?? "",
    filter.requestText ?? "",
    filter.requestedCapabilityId ?? "",
    filter.subjectId ?? "",
    filter.subjectSelector ?? "",
    filter.targetId,
    filter.templateId,
    filter.tenantId,
    filter.traceLimit ?? 0,
    filter.workspaceId
  ]);
}
