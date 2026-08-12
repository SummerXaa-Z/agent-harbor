import { useEffect, useRef, useState } from "react";

import {
  createAccessHandoffToken,
  fetchAccessHandoff,
  revokeAccessHandoffToken
} from "../api";
import type { Translator } from "../consolePresenters";
import type { Language } from "../i18n";
import {
  localizedErrorMessageState,
  localizedMessageText,
  type LocalizedMessage
} from "../localizedMessages";
import type {
  AccessHandoff,
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

  useEffect(() => {
    setOneTimeToken(null);
    if (!enabled || !accessHandoffFilterReady(filter)) {
      setHandoff(null);
      setMessage(null);
      return;
    }
    const controller = new AbortController();
    setLoading(true);
    setMessage(null);
    fetchAccessHandoff(filter, adminKey, controller.signal)
      .then(setHandoff)
      .catch((error) => {
        if (controller.signal.aborted) return;
        setHandoff(null);
        setMessage(localizedErrorMessageState(error, "error.loadAccessHandoff"));
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
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
    filter.subjectId ?? "",
    filter.subjectSelector ?? "",
    filter.targetId,
    filter.templateId,
    filter.tenantId,
    filter.traceLimit ?? 0,
    filter.workspaceId
  ]);
}
