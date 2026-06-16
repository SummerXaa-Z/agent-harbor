import { useCallback, useEffect, useState } from "react";

import {
  ApiRequestError,
  createAdminIdentity,
  disableAdminIdentity,
  fetchAdminIdentities,
  rotateAdminIdentityKey
} from "../api";
import {
  adminAccessReloadFailedMessageKey,
  reloadAfterAdminAccessMutation
} from "../adminAccessMutationReload";
import type {
  AdminIdentity,
  CreateAdminIdentityRequest
} from "../types";

export interface AdminAccessMessage {
  key: string;
  params?: Record<string, string | number>;
}

interface AdminAccessState {
  creating: boolean;
  forbidden: boolean;
  identities: AdminIdentity[];
  loading: boolean;
  message: AdminAccessMessage | null;
  modal: "create" | "rotate" | "disable" | null;
  oneTimeKey: string;
  selected: AdminIdentity | null;
}

interface LoadAdminIdentitiesOptions {
  throwOnError?: boolean;
}

const initialState: AdminAccessState = {
  creating: false,
  forbidden: false,
  identities: [],
  loading: false,
  message: null,
  modal: null,
  oneTimeKey: "",
  selected: null
};

export function useAdminAccessController({
  adminKey,
  enabled
}: {
  adminKey: string;
  enabled: boolean;
}) {
  const [state, setState] = useState<AdminAccessState>(initialState);

  const loadAdminIdentities = useCallback(async (signal?: AbortSignal, options: LoadAdminIdentitiesOptions = {}) => {
    setState((current) => ({ ...current, loading: true }));
    try {
      const identities = await fetchAdminIdentities(adminKey, signal);
      setState((current) => ({
        ...current,
        forbidden: false,
        identities,
        message: current.message?.key.startsWith("error.adminAccessLoad") ? null : current.message
      }));
    } catch (error) {
      if (isAbortError(error)) return;
      if (options.throwOnError) throw error;
      setState((current) => ({
        ...current,
        forbidden: isForbiddenAdminAccessError(error),
        identities: isForbiddenAdminAccessError(error) ? [] : current.identities,
        message: localizedAdminAccessError(error, "error.adminAccessLoad")
      }));
    } finally {
      setState((current) => ({ ...current, loading: false }));
    }
  }, [adminKey]);

  useEffect(() => {
    if (!enabled) return;
    const controller = new AbortController();
    void loadAdminIdentities(controller.signal);
    return () => controller.abort();
  }, [enabled, loadAdminIdentities]);

  function openCreate() {
    if (state.forbidden) {
      setState((current) => ({ ...current, message: { key: "error.adminAccessPlatformRequired" } }));
      return;
    }
    setState((current) => ({ ...current, modal: "create", selected: null }));
  }

  function openRotate(identity: AdminIdentity) {
    setState((current) => ({ ...current, modal: "rotate", selected: identity }));
  }

  function openDisable(identity: AdminIdentity) {
    setState((current) => ({ ...current, modal: "disable", selected: identity }));
  }

  function closeModal() {
    setState((current) => ({ ...current, modal: null, selected: null }));
  }

  async function submitCreate(body: CreateAdminIdentityRequest) {
    setState((current) => ({ ...current, creating: true, message: null, oneTimeKey: "" }));
    try {
      const created = await createAdminIdentity(body, adminKey);
      const actor = created.identity.displayName || created.identity.actor;
      setState((current) => ({
        ...current,
        identities: mergeIdentity(current.identities, created.identity),
        message: { key: "message.adminAccessCreated", params: { actor } },
        modal: null,
        oneTimeKey: created.key,
        selected: created.identity
      }));
      const reloadResult = await reloadAfterAdminAccessMutation({
        action: "create_admin",
        onReload: () => loadAdminIdentities(undefined, { throwOnError: true })
      });
      if (!reloadResult.ok) {
        setState((current) => ({
          ...current,
          message: { key: adminAccessReloadFailedMessageKey("create_admin"), params: { actor } }
        }));
      }
    } catch (error) {
      setState((current) => ({ ...current, message: localizedAdminAccessError(error, "error.adminAccessCreate") }));
    } finally {
      setState((current) => ({ ...current, creating: false }));
    }
  }

  async function submitRotate() {
    if (!state.selected) {
      setState((current) => ({ ...current, message: { key: "error.adminAccessSelectionRequired" } }));
      return;
    }
    const selected = state.selected;
    setState((current) => ({ ...current, creating: true, message: null, oneTimeKey: "" }));
    try {
      const rotated = await rotateAdminIdentityKey(selected.id, adminKey);
      const actor = rotated.identity.displayName || rotated.identity.actor;
      setState((current) => ({
        ...current,
        identities: mergeIdentity(current.identities, rotated.identity),
        message: { key: "message.adminAccessRotated", params: { actor } },
        modal: null,
        oneTimeKey: rotated.key,
        selected: rotated.identity
      }));
      const reloadResult = await reloadAfterAdminAccessMutation({
        action: "rotate_admin_key",
        onReload: () => loadAdminIdentities(undefined, { throwOnError: true })
      });
      if (!reloadResult.ok) {
        setState((current) => ({
          ...current,
          message: { key: adminAccessReloadFailedMessageKey("rotate_admin_key"), params: { actor } }
        }));
      }
    } catch (error) {
      setState((current) => ({ ...current, message: localizedAdminAccessError(error, "error.adminAccessRotate") }));
    } finally {
      setState((current) => ({ ...current, creating: false }));
    }
  }

  async function submitDisable() {
    if (!state.selected) {
      setState((current) => ({ ...current, message: { key: "error.adminAccessSelectionRequired" } }));
      return;
    }
    const selected = state.selected;
    setState((current) => ({ ...current, creating: true, message: null }));
    try {
      const disabled = await disableAdminIdentity(selected.id, adminKey);
      const actor = disabled.displayName || disabled.actor;
      setState((current) => ({
        ...current,
        identities: mergeIdentity(current.identities, disabled),
        message: { key: "message.adminAccessDisabled", params: { actor } },
        modal: null,
        selected: disabled
      }));
      const reloadResult = await reloadAfterAdminAccessMutation({
        action: "disable_admin",
        onReload: () => loadAdminIdentities(undefined, { throwOnError: true })
      });
      if (!reloadResult.ok) {
        setState((current) => ({
          ...current,
          message: { key: adminAccessReloadFailedMessageKey("disable_admin"), params: { actor } }
        }));
      }
    } catch (error) {
      setState((current) => ({ ...current, message: localizedAdminAccessError(error, "error.adminAccessDisable") }));
    } finally {
      setState((current) => ({ ...current, creating: false }));
    }
  }

  function clearOneTimeKey() {
    setState((current) => ({ ...current, oneTimeKey: "", selected: null }));
  }

  return {
    clearOneTimeKey,
    closeModal,
    loadAdminIdentities,
    openCreate,
    openDisable,
    openRotate,
    submitCreate,
    submitDisable,
    submitRotate,
    ...state
  };
}

export type AdminAccessController = ReturnType<typeof useAdminAccessController>;

function mergeIdentity(rows: AdminIdentity[], identity: AdminIdentity) {
  const found = rows.some((row) => row.id === identity.id);
  if (!found) return [identity, ...rows];
  return rows.map((row) => (row.id === identity.id ? identity : row));
}

function localizedAdminAccessError(error: unknown, fallbackKey: string): AdminAccessMessage {
  if (isForbiddenAdminAccessError(error)) {
    return { key: "error.adminAccessPlatformRequired" };
  }
  if (error instanceof Error && error.message.trim()) {
    return { key: "error.adminAccessOperation", params: { detail: error.message } };
  }
  return { key: fallbackKey };
}

function isAbortError(error: unknown) {
  return error instanceof Error && error.name === "AbortError";
}

function isForbiddenAdminAccessError(error: unknown) {
  return error instanceof ApiRequestError && error.status === 403;
}
