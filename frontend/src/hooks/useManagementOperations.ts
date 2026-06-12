import { useState, type FormEvent } from "react";
import {
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
import { parseRetryFields } from "../retryForm";
import type {
  Agent,
  AgentStatus,
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

interface UseManagementOperationsArgs {
  adminKey: string;
  defaultScope: ManagementScope;
  language: Language;
  onRefresh: () => Promise<void>;
  scope: ManagementScope;
  t: Translator;
}

export function useManagementOperations({
  adminKey,
  defaultScope,
  language,
  onRefresh,
  scope,
  t
}: UseManagementOperationsArgs) {
  const [agentForm, setAgentForm] = useState(defaultAgentForm);
  const [agentMessage, setAgentMessage] = useState("");
  const [keyForm, setKeyForm] = useState(defaultKeyForm);
  const [keyMessage, setKeyMessage] = useState("");
  const [createdKey, setCreatedKey] = useState<CreateAgentKeyResponse | null>(null);
  const [rotateForm, setRotateForm] = useState(defaultRotateForm);
  const [rotateMessage, setRotateMessage] = useState("");
  const [policyForm, setPolicyForm] = useState(defaultPolicyForm);
  const [policyMessage, setPolicyMessage] = useState("");
  const [cleanupActionId, setCleanupActionId] = useState("");

  async function submitAgent(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setAgentMessage("");
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
          setAgentMessage(t("message.validationCredentialGroup"));
          return;
        }
        channelConfig.credentialHeaders = { [credentialHeader]: credentialName };
        credentials = { [credentialName]: credentialValue };
      }
      const requestScope = normalizedScope(scope, defaultScope);
      await createAgent(
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
      setAgentForm(defaultAgentForm);
      setAgentMessage(t("message.agentCreated"));
      await onRefresh();
    } catch (error) {
      setAgentMessage(localizedErrorMessage(t, language, error, "error.createAgent"));
    }
  }

  async function handleAgentStatusChange(agent: Agent, status: AgentStatus) {
    setAgentMessage("");
    setCleanupActionId(agent.id);
    try {
      if (status === "disabled") {
        await disableAgent(agent.id, adminKey);
      } else {
        await updateAgent(agent.id, { status }, adminKey);
      }
      setAgentMessage(tx(t, "message.statusChanged", { name: agent.name, status: agentStatusLabel(status, t) }));
      await onRefresh();
    } catch (error) {
      setAgentMessage(localizedErrorMessage(t, language, error, "error.updateAgentStatus"));
    } finally {
      setCleanupActionId("");
    }
  }

  async function handleDisablePolicy(policy: RoutePolicy) {
    setPolicyMessage("");
    setCleanupActionId(policy.id);
    try {
      await disableRoutePolicy(policy.id, adminKey);
      setPolicyMessage(t("message.policyDisabled"));
      await onRefresh();
    } catch (error) {
      setPolicyMessage(localizedErrorMessage(t, language, error, "error.disableRoutePolicy"));
    } finally {
      setCleanupActionId("");
    }
  }

  async function submitKey(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setKeyMessage("");
    setCreatedKey(null);
    try {
      const ttl = Number(keyForm.expiresInSeconds);
      if (!Number.isInteger(ttl) || ttl < 1 || ttl > 3600) {
        setKeyMessage(t("message.validationTtl"));
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
      setKeyMessage(t("message.keyCreated"));
      setKeyForm({ ...defaultKeyForm, agentId: keyForm.agentId });
    } catch (error) {
      setKeyMessage(localizedErrorMessage(t, language, error, "error.createKey"));
    }
  }

  async function submitCredentialRotation(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setRotateMessage("");
    try {
      const credentialName = rotateForm.credentialName.trim();
      if (!rotateForm.agentId) {
        setRotateMessage(t("message.validationRotateAgent"));
        return;
      }
      if (!credentialName || !rotateForm.credentialValue.trim()) {
        setRotateMessage(t("message.validationCredentialRequired"));
        return;
      }
      await rotateAgentCredentials(
        rotateForm.agentId,
        { credentials: { [credentialName]: rotateForm.credentialValue } },
        adminKey
      );
      setRotateForm({ ...defaultRotateForm, agentId: rotateForm.agentId, credentialName });
      setRotateMessage(t("message.credentialRotated"));
      await onRefresh();
    } catch (error) {
      setRotateMessage(localizedErrorMessage(t, language, error, "error.rotateCredential"));
    }
  }

  async function submitRoutePolicy(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPolicyMessage("");
    try {
      const priority = Number(policyForm.priority);
      if (!Number.isInteger(priority) || priority < 0) {
        setPolicyMessage(t("message.validationPriority"));
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
      await createRoutePolicy(
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
      setPolicyMessage(t("message.policyCreated"));
      setPolicyForm({ ...defaultPolicyForm, callerAgentId: policyForm.callerAgentId });
      await onRefresh();
    } catch (error) {
      setPolicyMessage(localizedErrorMessage(t, language, error, "error.createRoutePolicy"));
    }
  }

  return {
    agentForm,
    agentMessage,
    cleanupActionId,
    createdKey,
    handleAgentStatusChange,
    handleDisablePolicy,
    keyForm,
    keyMessage,
    policyForm,
    policyMessage,
    rotateForm,
    rotateMessage,
    setAgentForm,
    setKeyForm,
    setPolicyForm,
    setRotateForm,
    submitAgent,
    submitCredentialRotation,
    submitKey,
    submitRoutePolicy
  };
}

function tx(t: Translator, key: string, values: Record<string, string | number>) {
  return Object.entries(values).reduce(
    (message, [name, value]) => message.replaceAll(`{${name}}`, String(value)),
    t(key)
  );
}

function localizedErrorMessage(t: Translator, language: Language, error: unknown, fallbackKey: string) {
  const fallback = t(fallbackKey);
  if (!(error instanceof Error) || !error.message.trim()) return fallback;
  if (language === "en" || /[\u4e00-\u9fa5]/.test(error.message)) {
    return error.message;
  }
  return fallback;
}

function retryFieldValidationMessage(message: string, t: Translator) {
  if (message === "Retry attempts must be an integer between 1 and 4.") {
    return t("message.validationRetryAttempts");
  }
  if (message === "Retry backoff must be an integer between 0 and 1000 ms.") {
    return t("message.validationRetryBackoff");
  }
  return message;
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
