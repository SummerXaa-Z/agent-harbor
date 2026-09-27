import { useCallback, useEffect, useRef, useState } from "react";
import {
  ApiRequestError,
  createAdminIdentity,
  disableAdminIdentity,
  fetchAdminIdentities,
  rotateAdminIdentityKey,
} from "../../api";
import type {
  AdminIdentity,
  CreateAdminIdentityRequest,
  CreateAdminIdentityResponse,
  RotateAdminIdentityKeyResponse,
} from "../../types";

export type AdminAccessAction = "create" | "rotate" | "disable";

export interface AdminAccessActionResult {
  ok: boolean;
  // The create/rotate response object; views extract the one-time secret,
  // mask it, and only ever pass it to the clipboard — never render it.
  outcome?: CreateAdminIdentityResponse | RotateAdminIdentityKeyResponse | AdminIdentity;
  // Raw failure so the view can surface the API's own message instead of a
  // generic "operation failed" dead end.
  error?: unknown;
}

export interface AdminAccessState {
  act: (action: AdminAccessAction, input: { body?: CreateAdminIdentityRequest; id?: string }) => Promise<AdminAccessActionResult>;
  failed: boolean;
  forbidden: boolean;
  identities: readonly AdminIdentity[];
  loading: boolean;
  reload: () => Promise<void>;
}

function isForbidden(error: unknown): boolean {
  return error instanceof ApiRequestError && error.status === 403;
}

// Named-admin management. Only platform admins may manage identities; a 403
// flips the view to its read-only state instead of an error dead end.
export function useAdminAccess(live: boolean): AdminAccessState {
  const [identities, setIdentities] = useState<readonly AdminIdentity[]>([]);
  const [loading, setLoading] = useState(false);
  const [failed, setFailed] = useState(false);
  const [forbidden, setForbidden] = useState(false);
  const runIdRef = useRef(0);

  const reload = useCallback(async () => {
    const runId = ++runIdRef.current;
    setLoading(true);
    try {
      const rows = await fetchAdminIdentities(undefined, undefined);
      if (runId !== runIdRef.current) return;
      setIdentities(rows);
      setFailed(false);
      setForbidden(false);
    } catch (error) {
      if (runId !== runIdRef.current) return;
      setForbidden(isForbidden(error));
      setFailed(!isForbidden(error));
    } finally {
      if (runId === runIdRef.current) setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!live) return;
    void reload();
  }, [live, reload]);

  const act = useCallback<AdminAccessState["act"]>(async (action, { body, id }) => {
    try {
      if (action === "create" && body) {
        const created = await createAdminIdentity(body);
        await reload();
        return { ok: true, outcome: created };
      }
      if (action === "rotate" && id) {
        const rotated = await rotateAdminIdentityKey(id);
        await reload();
        return { ok: true, outcome: rotated };
      }
      if (action === "disable" && id) {
        const identity = await disableAdminIdentity(id);
        await reload();
        return { ok: true, outcome: identity };
      }
      return { ok: false };
    } catch (error) {
      return { ok: false, error };
    }
  }, [reload]);

  return { act, failed, forbidden, identities, loading, reload };
}
