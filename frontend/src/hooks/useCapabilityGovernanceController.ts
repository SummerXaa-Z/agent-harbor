import { useEffect, useState, type Dispatch, type FormEvent, type SetStateAction } from "react";
import {
  createInstanceAssignment,
  createTenantEntitlement,
  createWorkspaceAssignment,
  refreshTargetCapabilities,
  updateCapability
} from "../api";
import {
  capabilityGrantRefreshFailedMessageKey,
  mergeCapabilityGrantChainIntoConsoleData,
  refreshAfterCapabilityGrantMutation
} from "../capabilityGrantRefresh";
import {
  appendLocalCapabilityGrantChain,
  defaultCapabilityGrantForm,
  mergeCapabilitiesForTarget,
  normalizeCapabilityGrantForm,
  shouldUseLocalCapabilityFallback,
  validateCapabilityGrantChain,
  type CapabilityGrantForm
} from "../capabilityGrantChain";
import {
  capabilityDisplayName,
  type Translator
} from "../consolePresenters";
import type { Language } from "../i18n";
import {
  localizedErrorMessageState,
  localizedMessageText,
  localizedUpstreamErrorMessageState,
  tx,
  type LocalizedMessage
} from "../localizedMessages";
import type {
  Capability,
  ConsoleData,
  ManagementScope
} from "../types";

interface UseCapabilityGovernanceControllerArgs {
  adminKey: string;
  data: ConsoleData | null;
  defaultScope: ManagementScope;
  language: Language;
  onRefresh: () => Promise<void>;
  setData: Dispatch<SetStateAction<ConsoleData | null>>;
  t: Translator;
}

export function useCapabilityGovernanceController({
  adminKey,
  data,
  defaultScope,
  language,
  onRefresh,
  setData,
  t
}: UseCapabilityGovernanceControllerArgs) {
  const [form, setForm] = useState<CapabilityGrantForm>(() => defaultCapabilityGrantForm(defaultScope));
  const [messageState, setMessage] = useState<LocalizedMessage | null>(null);
  const [actionId, setActionId] = useState("");
  const message = localizedMessageText(messageState, t, language);

  useEffect(() => {
    if (!data) return;
    setForm((current) => normalizeCapabilityGrantForm(current, data));
  }, [data]);

  async function handleRefreshTargetCapabilities() {
    const targetId = form.targetId.trim();
    if (!targetId) {
      setMessage({ key: "message.validationMcpTargetRequired" });
      return;
    }
    setMessage(null);
    setActionId(`refresh:${targetId}`);
    try {
      const refreshed = await refreshTargetCapabilities(targetId, adminKey);
      setData((current) =>
        current
          ? {
              ...current,
              capabilities: mergeCapabilitiesForTarget(current.capabilities, refreshed, targetId),
              capabilitiesLoadedFromApi: true
            }
          : current
      );
      setMessage({ key: "message.refreshedCapabilities", params: { count: refreshed.length } });
    } catch (error) {
      if (shouldUseLocalCapabilityFallback(error, data)) {
        setMessage({ key: "message.capabilityFallback" });
        return;
      }
      setMessage(localizedUpstreamErrorMessageState(error, "error.refreshCapabilities"));
    } finally {
      setActionId("");
    }
  }

  async function handleApproveCapability(capability: Capability) {
    setMessage(null);
    setActionId(capability.id);
    const needsDataDomain = (capability.dataDomains?.length ?? 0) === 0;
    try {
      const updated = await updateCapability(capability.id, { discoveryStatus: "approved" }, adminKey);
      setData((current) =>
        current
          ? {
              ...current,
              capabilities: current.capabilities.map((item) => (item.id === updated.id ? updated : item)),
              capabilitiesLoadedFromApi: true
            }
          : current
      );
      setMessage(
        needsDataDomain
          ? {
              render: (t) => tx(t, "message.capabilityApprovedNeedsDataDomain", { name: capabilityDisplayName(capability, t) })
            }
          : {
              render: (t) => tx(t, "message.capabilityApproved", { name: capabilityDisplayName(capability, t) })
            }
      );
    } catch (error) {
      if (shouldUseLocalCapabilityFallback(error, data)) {
        setData((current) =>
          current
            ? {
                ...current,
                capabilities: current.capabilities.map((item) =>
                  item.id === capability.id
                    ? { ...item, discoveryStatus: "approved", updatedAt: new Date().toISOString() }
                    : item
                )
              }
            : current
        );
        setMessage(
          needsDataDomain
            ? {
                render: (t) => tx(t, "message.capabilityApprovedNeedsDataDomain", { name: capabilityDisplayName(capability, t) })
              }
            : {
                render: (t) => tx(t, "message.capabilityApprovedFallback", { name: capabilityDisplayName(capability, t) })
              }
        );
        return;
      }
      setMessage(localizedErrorMessageState(error, "error.approveCapability"));
    } finally {
      setActionId("");
    }
  }

  async function handleClassifyCapability(capability: Capability, dataDomain: string) {
    const normalizedDomain = dataDomain.trim();
    if (!normalizedDomain) {
      setMessage({ key: "message.validationCapabilityDataDomainRequired" });
      return;
    }
    setMessage(null);
    setActionId(`classify:${capability.id}`);
    try {
      const updated = await updateCapability(capability.id, { dataDomains: [normalizedDomain] }, adminKey);
      setData((current) => current ? {
        ...current,
        capabilities: current.capabilities.map((item) => item.id === updated.id ? updated : item),
        capabilitiesLoadedFromApi: true
      } : current);
      setMessage({
        render: (t) => tx(t, "message.capabilityDataDomainSaved", {
          domain: normalizedDomain,
          name: capabilityDisplayName(capability, t)
        })
      });
    } catch (error) {
      setMessage(localizedErrorMessageState(error, "error.classifyCapability"));
    } finally {
      setActionId("");
    }
  }

  async function submitCapabilityGrantChain(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage(null);
    const outcome = validateCapabilityGrantChain(form, data?.capabilities ?? []);
    if ("messageKey" in outcome) {
      setMessage({ key: outcome.messageKey });
      return;
    }
    const { capability, dataScopes, tenantId, workspaceId, callerInstanceId, subjectSelector } = outcome.plan;
    setActionId(`grant:${capability.id}`);
    try {
      const entitlement = await createTenantEntitlement(
        {
          capabilityId: capability.id,
          dataScopes,
          effect: "allow",
          priority: 50,
          status: "enabled",
          targetId: capability.targetId,
          tenantId
        },
        adminKey
      );
      const workspaceAssignment = await createWorkspaceAssignment(
        {
          dataScopes,
          effect: "allow",
          status: "enabled",
          tenantEntitlementId: entitlement.id,
          workspaceId
        },
        adminKey
      );
      const instanceAssignment = await createInstanceAssignment(
        {
          callerInstanceId,
          dataScopes,
          effect: "allow",
          status: "enabled",
          subjectSelector,
          workspaceAssignmentId: workspaceAssignment.id
        },
        adminKey
      );
      setData((current) =>
        current
          ? mergeCapabilityGrantChainIntoConsoleData(current, {
              entitlement,
              instanceAssignment,
              workspaceAssignment
            })
          : current
      );
      setMessage({ key: "message.grantChainCreated" });
      const refreshResult = await refreshAfterCapabilityGrantMutation({ onRefresh });
      if (!refreshResult.ok) {
        setMessage({ key: capabilityGrantRefreshFailedMessageKey() });
      }
    } catch (error) {
      if (shouldUseLocalCapabilityFallback(error, data) && data) {
        setData((current) =>
          current ? appendLocalCapabilityGrantChain(current, capability, form, dataScopes, defaultScope) : current
        );
        setMessage({ key: "message.grantChainCreatedFallback" });
        return;
      }
      setMessage(localizedErrorMessageState(error, "error.createGrantChain"));
    } finally {
      setActionId("");
    }
  }

  return {
    actionId,
    form,
    handleApproveCapability,
    handleClassifyCapability,
    handleRefreshTargetCapabilities,
    message,
    setForm,
    submitCapabilityGrantChain
  };
}

