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
import type { CapabilityGrantForm } from "../components/CapabilityGovernanceView";
import {
  capabilityDisplayName,
  type Translator
} from "../consolePresenters";
import type { Language } from "../i18n";
import {
  localizedErrorMessageState,
  localizedMessageText,
  tx,
  type LocalizedMessage
} from "../localizedMessages";
import type {
  Agent,
  Capability,
  ConsoleData,
  DataScope,
  ManagementScope,
  TenantEntitlement,
  WorkspaceAssignment
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
  const [form, setForm] = useState<CapabilityGrantForm>(() => ({
    callerInstanceId: "",
    capabilityId: "",
    subjectSelector: "user:support-*",
    targetId: "",
    tenantId: defaultScope.tenantId,
    workspaceId: defaultScope.workspaceId
  }));
  const [messageState, setMessage] = useState<LocalizedMessage | null>(null);
  const [actionId, setActionId] = useState("");
  const message = localizedMessageText(messageState, t, language);

  useEffect(() => {
    if (!data) return;
    setForm((current) => {
      const mcpTarget = data.agents.find((agent) => agent.channelType === "mcp");
      const targetId = current.targetId || mcpTarget?.id || "";
      const capability = data.capabilities.find((item) => item.id === current.capabilityId && item.targetId === targetId)
        ?? data.capabilities.find((item) => item.targetId === targetId)
        ?? data.capabilities[0];
      const caller = data.agents.find(
        (agent) =>
          agent.status === "active" &&
          agent.tenantId === current.tenantId &&
          agent.workspaceId === current.workspaceId &&
          agent.channelType === "local"
      ) ?? data.agents.find((agent) => agent.status === "active" && agent.channelType === "local");
      const next = {
        ...current,
        callerInstanceId: current.callerInstanceId || caller?.id || "",
        capabilityId: current.capabilityId || capability?.id || "",
        targetId
      };
      return shallowEqualCapabilityForm(current, next) ? current : next;
    });
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
      setMessage(localizedErrorMessageState(error, "error.refreshCapabilities"));
    } finally {
      setActionId("");
    }
  }

  async function handleApproveCapability(capability: Capability) {
    setMessage(null);
    setActionId(capability.id);
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
      setMessage({
        render: (t) => tx(t, "message.capabilityApproved", { name: capabilityDisplayName(capability, t) })
      });
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
        setMessage({
          render: (t) => tx(t, "message.capabilityApprovedFallback", { name: capabilityDisplayName(capability, t) })
        });
        return;
      }
      setMessage(localizedErrorMessageState(error, "error.approveCapability"));
    } finally {
      setActionId("");
    }
  }

  async function submitCapabilityGrantChain(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage(null);
    const capability = data?.capabilities.find((item) => item.id === form.capabilityId);
    const tenantId = form.tenantId.trim();
    const workspaceId = form.workspaceId.trim();
    const callerInstanceId = form.callerInstanceId.trim();
    if (!capability) {
      setMessage({ key: "message.validationCapabilityRequired" });
      return;
    }
    if (!tenantId || !workspaceId || !callerInstanceId) {
      setMessage({ key: "message.validationTenantWorkspaceCaller" });
      return;
    }
    const subjectSelector = form.subjectSelector.trim();
    if (!subjectSelector || subjectSelector === "*") {
      setMessage({ key: "message.validationSubjectSelectorRequired" });
      return;
    }
    const dataScopes = capability.dataScopes ?? [];
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
    handleRefreshTargetCapabilities,
    message,
    setForm,
    submitCapabilityGrantChain
  };
}

function shallowEqualCapabilityForm(left: CapabilityGrantForm, right: CapabilityGrantForm) {
  return (
    left.callerInstanceId === right.callerInstanceId &&
    left.capabilityId === right.capabilityId &&
    left.subjectSelector === right.subjectSelector &&
    left.targetId === right.targetId &&
    left.tenantId === right.tenantId &&
    left.workspaceId === right.workspaceId
  );
}

function mergeCapabilitiesForTarget(existing: Capability[], refreshed: Capability[], targetId: string) {
  return [
    ...existing.filter((capability) => capability.targetId !== targetId),
    ...refreshed
  ];
}

function shouldUseLocalCapabilityFallback(error: unknown, data: ConsoleData | null) {
  if (data?.capabilitiesLoadedFromApi) return false;
  return error instanceof Error && /Failed to fetch|404|not found|Not Found/i.test(error.message);
}

function appendLocalCapabilityGrantChain(
  current: ConsoleData,
  capability: Capability,
  form: CapabilityGrantForm,
  dataScopes: DataScope[],
  defaultScope: ManagementScope
): ConsoleData {
  const now = new Date().toISOString();
  const tenantId = form.tenantId.trim() || defaultScope.tenantId;
  const workspaceId = form.workspaceId.trim() || defaultScope.workspaceId;
  const callerInstanceId = form.callerInstanceId.trim();
  const subjectSelector = form.subjectSelector.trim();

  const existingEntitlement = current.tenantEntitlements.find(
    (item) => item.tenantId === tenantId && item.targetId === capability.targetId && item.capabilityId === capability.id
  );
  const entitlement: TenantEntitlement = existingEntitlement
    ? { ...existingEntitlement, dataScopes, effect: "allow", status: "enabled", updatedAt: now }
    : {
        capabilityId: capability.id,
        createdAt: now,
        dataScopes,
        effect: "allow",
        id: nextLocalId("tent", [tenantId, capability.id], current.tenantEntitlements.map((item) => item.id)),
        priority: 50,
        status: "enabled",
        targetId: capability.targetId,
        tenantId,
        updatedAt: now
      };
  const tenantEntitlements = existingEntitlement
    ? current.tenantEntitlements.map((item) => (item.id === entitlement.id ? entitlement : item))
    : [entitlement, ...current.tenantEntitlements];

  const existingWorkspaceAssignment = current.workspaceAssignments.find(
    (item) => item.tenantEntitlementId === entitlement.id && item.workspaceId === workspaceId
  );
  const workspaceAssignment: WorkspaceAssignment = existingWorkspaceAssignment
    ? { ...existingWorkspaceAssignment, dataScopes, effect: "allow", status: "enabled", updatedAt: now }
    : {
        createdAt: now,
        dataScopes,
        effect: "allow",
        id: nextLocalId("wsa", [workspaceId, entitlement.id], current.workspaceAssignments.map((item) => item.id)),
        status: "enabled",
        tenantId,
        tenantEntitlementId: entitlement.id,
        updatedAt: now,
        workspaceId
      };
  const workspaceAssignments = existingWorkspaceAssignment
    ? current.workspaceAssignments.map((item) => (item.id === workspaceAssignment.id ? workspaceAssignment : item))
    : [workspaceAssignment, ...current.workspaceAssignments];

  const existingInstanceAssignment = current.instanceAssignments.find(
    (item) => item.workspaceAssignmentId === workspaceAssignment.id && item.callerInstanceId === callerInstanceId
  );
  const instanceAssignment = existingInstanceAssignment
    ? { ...existingInstanceAssignment, dataScopes, effect: "allow" as const, status: "enabled" as const, subjectSelector, updatedAt: now }
    : {
        callerInstanceId,
        createdAt: now,
        dataScopes,
        effect: "allow" as const,
        id: nextLocalId("ia", [callerInstanceId, workspaceAssignment.id], current.instanceAssignments.map((item) => item.id)),
        status: "enabled" as const,
        subjectSelector,
        tenantId,
        updatedAt: now,
        workspaceId,
        workspaceAssignmentId: workspaceAssignment.id
      };
  const instanceAssignments = existingInstanceAssignment
    ? current.instanceAssignments.map((item) => (item.id === instanceAssignment.id ? instanceAssignment : item))
    : [instanceAssignment, ...current.instanceAssignments];

  return {
    ...current,
    capabilities: current.capabilities.map((item) =>
      item.id === capability.id ? { ...item, discoveryStatus: "approved", updatedAt: now } : item
    ),
    instanceAssignments,
    tenantEntitlements,
    workspaceAssignments
  };
}

function nextLocalId(prefix: string, parts: string[], existing: string[]) {
  const base = `${prefix}_${parts.map(safeIdPart).filter(Boolean).join("_") || "local"}`;
  let candidate = base;
  let index = 1;
  while (existing.includes(candidate)) {
    index += 1;
    candidate = `${base}_${index}`;
  }
  return candidate;
}

function safeIdPart(value: string) {
  return value.trim().replace(/[^a-zA-Z0-9_-]+/g, "_").replace(/^_+|_+$/g, "").slice(0, 40);
}
