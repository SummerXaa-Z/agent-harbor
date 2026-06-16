import { useRef, useState, type FormEvent } from "react";
import {
  ApiRequestError,
  createAgent,
  createAgentKey,
  createRoutePolicy,
  disableAgent,
  disableRoutePolicy,
  rotateAgentCredentials,
  updateAgent
} from "../api";
import type {
  AgentCreateFormState,
  CredentialRotateFormState,
  KeyCreateFormState,
  PolicyCreateFormState
} from "../components/ManagementForms";
import type { Translator } from "../consolePresenters";
import type { Language } from "../i18n";
import {
  localizedErrorMessage,
  localizedErrorMessageState,
  localizedMessageText,
  tx,
  type LocalizedMessage
} from "../localizedMessages";
import {
  mergeManagementAgentIntoConsoleData,
  mergeManagementRoutePolicyIntoConsoleData
} from "../managementLocalData";
import {
  managementMutationRefreshFailedMessageKey,
  managementMutationSuccessMessageKey,
  refreshAfterManagementMutation,
  type ManagementMutationAction,
  type ManagementMutationRefreshState
} from "../managementMutationRefresh";
import { parseRetryFields } from "../retryForm";
import type {
  Agent,
  AgentStatus,
  ConsoleData,
  CreateAgentKeyResponse,
  JsonObject,
  ManagementScope,
  RoutePolicy
} from "../types";

const defaultAgentForm: AgentCreateFormState = {
  channelType: "local",
  credentialHeader: "",
  credentialName: "",
  credentialValue: "",
  description: "",
  endpoint: "",
  name: "",
  retryBackoffMs: "0",
  retryMaxAttempts: "1",
  status: "draft" as AgentStatus
};
const defaultKeyForm: KeyCreateFormState = { agentId: "", expiresInSeconds: "900", name: "console key" };
const defaultRotateForm: CredentialRotateFormState = { agentId: "", credentialName: "apiToken", credentialValue: "" };
const defaultPolicyForm: PolicyCreateFormState = {
  callerAgentId: "",
  effect: "allow",
  name: "",
  priority: "100",
  retryBackoffMs: "0",
  retryMaxAttempts: "1",
  routeKey: "",
  routeType: "mcp",
  targetAgentId: ""
};

type ActiveManagementMutationAction = "" | ManagementMutationAction;

interface UseManagementOperationsArgs {
  adminKey: string;
  defaultScope: ManagementScope;
  language: Language;
  onDataPatch?: (updater: (current: ConsoleData) => ConsoleData) => void;
  onRefresh: () => Promise<void>;
  scope: ManagementScope;
  t: Translator;
}

export function useManagementOperations({
  adminKey,
  defaultScope,
  language,
  onDataPatch,
  onRefresh,
  scope,
  t
}: UseManagementOperationsArgs) {
  const [agentForm, setAgentForm] = useState(defaultAgentForm);
  const [agentMessageState, setAgentMessage] = useState<LocalizedMessage | null>(null);
  const [keyForm, setKeyForm] = useState(defaultKeyForm);
  const [keyMessageState, setKeyMessage] = useState<LocalizedMessage | null>(null);
  const [createdKey, setCreatedKey] = useState<CreateAgentKeyResponse | null>(null);
  const [rotateForm, setRotateForm] = useState(defaultRotateForm);
  const [rotateMessageState, setRotateMessage] = useState<LocalizedMessage | null>(null);
  const [policyForm, setPolicyForm] = useState(defaultPolicyForm);
  const [policyMessageState, setPolicyMessage] = useState<LocalizedMessage | null>(null);
  const [cleanupActionId, setCleanupActionId] = useState("");
  const managementMutationInFlightRef = useRef<ActiveManagementMutationAction>("");
  const [managementMutationAction, setManagementMutationAction] = useState<ActiveManagementMutationAction>("");
  const [managementRefreshState, setManagementRefreshState] = useState<ManagementMutationRefreshState>({ status: "idle" });
  const agentMessage = localizedMessageText(agentMessageState, t, language);
  const keyMessage = localizedMessageText(keyMessageState, t, language);
  const rotateMessage = localizedMessageText(rotateMessageState, t, language);
  const policyMessage = localizedMessageText(policyMessageState, t, language);

  function beginManagementMutation(action: ManagementMutationAction) {
    if (managementMutationInFlightRef.current) return false;
    managementMutationInFlightRef.current = action;
    setManagementMutationAction(action);
    return true;
  }

  function endManagementMutation(action: ManagementMutationAction) {
    if (managementMutationInFlightRef.current !== action) return;
    managementMutationInFlightRef.current = "";
    setManagementMutationAction("");
  }

  function clearCreatedKey() {
    setCreatedKey(null);
  }

  function updateKeyForm(next: KeyCreateFormState) {
    setCreatedKey(null);
    setKeyForm(next);
  }

  function patchConsoleData(updater: (current: ConsoleData) => ConsoleData) {
    onDataPatch?.(updater);
  }

  async function finishManagementMutation(
    action: ManagementMutationAction,
    setMessage: (message: LocalizedMessage | null) => void,
    successMessage?: LocalizedMessage
  ) {
    setManagementRefreshState({ action, status: "refreshing" });
    const refreshResult = await refreshAfterManagementMutation({ action, onRefresh });
    if (refreshResult.ok) {
      setManagementRefreshState({ action, refreshedAt: refreshResult.refreshedAt, status: "fresh" });
      setMessage(successMessage ?? { key: managementMutationSuccessMessageKey(action) });
      return;
    }
    setManagementRefreshState({
      action,
      errorMessage: localizedErrorMessage(t, language, refreshResult.error, "error.refreshManagementData"),
      status: "stale"
    });
    setMessage({ key: managementMutationRefreshFailedMessageKey(action) });
  }

  async function submitAgent(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!beginManagementMutation("create_agent")) return;
    setAgentMessage(null);
    try {
      const channelConfig: JsonObject = {};
      const endpoint = agentForm.endpoint.trim();
      if (endpoint) channelConfig.endpoint = endpoint;
      const retry = parseRetryFields({
        backoffMsText: agentForm.retryBackoffMs,
        maxAttemptsText: agentForm.retryMaxAttempts
      });
      if (!retry.ok) {
        setAgentMessage(retryFieldValidationMessage(retry.message, t));
        return;
      }
      if (retry.requested) {
        channelConfig.retry = {
          backoffMs: retry.backoffMs,
          maxAttempts: retry.maxAttempts
        };
      }
      const credentialHeader = agentForm.credentialHeader.trim();
      const credentialName = agentForm.credentialName.trim();
      const credentialValue = agentForm.credentialValue;
      const hasCredentialInput = Boolean(credentialHeader || credentialName || credentialValue.trim());
      let credentials: Record<string, string> | undefined;
      if (hasCredentialInput) {
        if (!credentialHeader || !credentialName || !credentialValue.trim()) {
          setAgentMessage({ key: "message.validationCredentialGroup" });
          return;
        }
        channelConfig.credentialHeaders = { [credentialHeader]: credentialName };
        credentials = { [credentialName]: credentialValue };
      }
      const requestScope = normalizedScope(scope, defaultScope);
      const created = await createAgent(
        {
          channelConfig: Object.keys(channelConfig).length > 0 ? channelConfig : undefined,
          channelType: agentForm.channelType.trim() || "local",
          credentials,
          description: agentForm.description.trim() || undefined,
          name: agentForm.name.trim(),
          status: agentForm.status,
          tenantId: requestScope.tenantId,
          workspaceId: requestScope.workspaceId
        },
        adminKey
      );
      patchConsoleData((current) => mergeManagementAgentIntoConsoleData(current, created));
      setAgentForm(defaultAgentForm);
      await finishManagementMutation("create_agent", setAgentMessage);
    } catch (error) {
      setAgentMessage(localizedManagementErrorMessageState(error, "error.createAgent"));
    } finally {
      endManagementMutation("create_agent");
    }
  }

  async function handleAgentStatusChange(agent: Agent, status: AgentStatus) {
    setAgentMessage(null);
    setCleanupActionId(agent.id);
    try {
      const successMessage = statusChangedMessage(agent.name, status);
      const updated = status === "disabled"
        ? await disableAgent(agent.id, adminKey)
        : await updateAgent(agent.id, { status }, adminKey);
      patchConsoleData((current) => mergeManagementAgentIntoConsoleData(current, updated));
      await finishManagementMutation("update_agent_status", setAgentMessage, successMessage);
    } catch (error) {
      setAgentMessage(localizedManagementErrorMessageState(error, "error.updateAgentStatus"));
    } finally {
      setCleanupActionId("");
    }
  }

  async function handleDisablePolicy(policy: RoutePolicy) {
    setPolicyMessage(null);
    setCleanupActionId(policy.id);
    try {
      const disabled = await disableRoutePolicy(policy.id, adminKey);
      patchConsoleData((current) => mergeManagementRoutePolicyIntoConsoleData(current, disabled));
      await finishManagementMutation("disable_policy", setPolicyMessage, { key: "message.policyDisabled" });
    } catch (error) {
      setPolicyMessage(localizedManagementErrorMessageState(error, "error.disableRoutePolicy"));
    } finally {
      setCleanupActionId("");
    }
  }

  async function submitKey(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setCreatedKey(null);
    if (!beginManagementMutation("create_key")) return;
    setKeyMessage(null);
    try {
      const ttl = Number(keyForm.expiresInSeconds);
      if (!Number.isInteger(ttl) || ttl < 1 || ttl > 3600) {
        setKeyMessage({ key: "message.validationTtl" });
        return;
      }
      const next = await createAgentKey(
        {
          agentId: keyForm.agentId,
          expiresInSeconds: ttl,
          name: keyForm.name.trim() || undefined
        },
        adminKey
      );
      setCreatedKey(next);
      setKeyForm({ ...defaultKeyForm, agentId: keyForm.agentId });
      await finishManagementMutation("create_key", setKeyMessage);
    } catch (error) {
      setKeyMessage(localizedManagementErrorMessageState(error, "error.createKey"));
    } finally {
      endManagementMutation("create_key");
    }
  }

  async function submitCredentialRotation(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!beginManagementMutation("rotate_credential")) return;
    setRotateMessage(null);
    try {
      const credentialName = rotateForm.credentialName.trim();
      if (!rotateForm.agentId) {
        setRotateMessage({ key: "message.validationRotateAgent" });
        return;
      }
      if (!credentialName || !rotateForm.credentialValue.trim()) {
        setRotateMessage({ key: "message.validationCredentialRequired" });
        return;
      }
      const updated = await rotateAgentCredentials(
        rotateForm.agentId,
        { credentials: { [credentialName]: rotateForm.credentialValue } },
        adminKey
      );
      patchConsoleData((current) => mergeManagementAgentIntoConsoleData(current, updated));
      setRotateForm({ ...defaultRotateForm, agentId: rotateForm.agentId, credentialName });
      await finishManagementMutation("rotate_credential", setRotateMessage);
    } catch (error) {
      setRotateMessage(localizedManagementErrorMessageState(error, "error.rotateCredential"));
    } finally {
      endManagementMutation("rotate_credential");
    }
  }

  async function submitRoutePolicy(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!beginManagementMutation("create_policy")) return;
    setPolicyMessage(null);
    try {
      const priority = Number(policyForm.priority);
      if (!Number.isInteger(priority) || priority < 0) {
        setPolicyMessage({ key: "message.validationPriority" });
        return;
      }
      const retry = parseRetryFields({
        backoffMsText: policyForm.retryBackoffMs,
        maxAttemptsText: policyForm.retryMaxAttempts
      });
      if (!retry.ok) {
        setPolicyMessage(retryFieldValidationMessage(retry.message, t));
        return;
      }
      const created = await createRoutePolicy(
        {
          callerAgentId: policyForm.callerAgentId,
          effect: policyForm.effect as "allow" | "deny",
          name: policyForm.name.trim() || undefined,
          priority,
          retry: retry.requested
            ? { backoffMs: retry.backoffMs, maxAttempts: retry.maxAttempts, statusCodes: [502, 503, 504] }
            : undefined,
          routeKey: policyForm.routeKey.trim() || undefined,
          routeType: policyForm.routeType.trim(),
          targetAgentId: policyForm.targetAgentId
        },
        adminKey
      );
      patchConsoleData((current) => mergeManagementRoutePolicyIntoConsoleData(current, created));
      setPolicyForm({ ...defaultPolicyForm, callerAgentId: policyForm.callerAgentId });
      await finishManagementMutation("create_policy", setPolicyMessage);
    } catch (error) {
      setPolicyMessage(localizedManagementErrorMessageState(error, "error.createRoutePolicy"));
    } finally {
      endManagementMutation("create_policy");
    }
  }

  return {
    agentForm,
    agentMessage,
    cleanupActionId,
    clearCreatedKey,
    createdKey,
    handleAgentStatusChange,
    handleDisablePolicy,
    keyForm,
    keyMessage,
    managementMutationAction,
    managementRefreshState,
    policyForm,
    policyMessage,
    rotateForm,
    rotateMessage,
    setAgentForm,
    setKeyForm: updateKeyForm,
    setPolicyForm,
    setRotateForm,
    submitAgent,
    submitCredentialRotation,
    submitKey,
    submitRoutePolicy
  };
}

function localizedManagementErrorMessageState(error: unknown, fallbackKey: string): LocalizedMessage {
  if (error instanceof ApiRequestError && error.code === "DUPLICATE_RESOURCE_MUTATION") {
    return { key: "message.duplicateResourceMutation" };
  }
  return localizedErrorMessageState(error, fallbackKey);
}

function retryFieldValidationMessage(message: string, t: Translator): LocalizedMessage {
  if (message === "Retry attempts must be an integer between 1 and 4.") {
    return { key: "message.validationRetryAttempts" };
  }
  if (message === "Retry backoff must be an integer between 0 and 1000 ms.") {
    return { key: "message.validationRetryBackoff" };
  }
  return { key: "message.validationRetryInvalid" };
}

function normalizedScope(scope: ManagementScope, defaultScope: ManagementScope): ManagementScope {
  return {
    tenantId: scope.tenantId.trim() || defaultScope.tenantId,
    workspaceId: scope.workspaceId.trim() || defaultScope.workspaceId
  };
}

function agentStatusLabel(status: AgentStatus, t: Translator) {
  if (status === "active") return t("status.agentActive");
  if (status === "disabled") return t("status.agentDisabled");
  return t("status.agentDraft");
}

function statusChangedMessage(name: string, status: AgentStatus): LocalizedMessage {
  return {
    render: (t) => tx(t, "message.statusChanged", { name, status: agentStatusLabel(status, t) })
  };
}
