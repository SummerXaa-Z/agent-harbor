import { useState } from "react";
import {
  Copy,
  KeyRound,
  RefreshCw,
  ShieldCheck,
  Trash2,
  X
} from "lucide-react";

import {
  dataScopeText,
  formatDate,
  readableIdentifierLabel,
  type Translator
} from "../consolePresenters";
import type { Language } from "../i18n";
import type {
  AccessHandoff,
  CreateAccessHandoffTokenResponse,
  PermissionPackageProductionReadinessFilter
} from "../permissionPackages";
import { tx } from "../localizedMessages";
import { useAccessHandoffController } from "../hooks/useAccessHandoffController";
import { Badge } from "./ui";

export function AccessHandoffPanel({
  adminKey,
  enabled,
  filter,
  language,
  refreshKey,
  t
}: {
  adminKey: string;
  enabled: boolean;
  filter: PermissionPackageProductionReadinessFilter;
  language: Language;
  refreshKey?: string;
  t: Translator;
}) {
  const controller = useAccessHandoffController({ adminKey, enabled, filter, language, refreshKey, t });
  const { handoff, loading, message, oneTimeToken, tokenAction } = controller;
  const [copied, setCopied] = useState("");
  const status = handoff?.status ?? "blocked";
  const ready = status === "ready";

  async function copyValue(key: string, value: string) {
    if (!value) return;
    try {
      await navigator.clipboard.writeText(value);
      setCopied(key);
      window.setTimeout(() => setCopied((current) => current === key ? "" : current), 1800);
    } catch {
      setCopied("");
    }
  }

  function revokeToken(id: string) {
    if (!window.confirm(t("accessHandoff.tokenRevokeConfirm"))) return;
    void controller.revokeToken(id);
  }

  return (
    <section className="access-handoff" aria-label={t("accessHandoff.title")}>
      <div className="access-handoff-header">
        <div>
          <span className="section-kicker">{t("accessHandoff.kicker")}</span>
          <h3>{t("accessHandoff.title")}</h3>
          <p>{ready ? t("accessHandoff.readyDetail") : t("accessHandoff.blockedDetail")}</p>
        </div>
        <div className="access-handoff-header-actions">
          <Badge tone={ready ? "success" : status === "needs_review" ? "warning" : "danger"}>
            {t(`accessHandoff.status.${status}`)}
          </Badge>
          <button className="secondary-button" disabled={loading} onClick={() => void controller.refresh()} type="button">
            <RefreshCw className={loading ? "spin" : ""} size={14} />
            {t("action.refreshAccessHandoff")}
          </button>
        </div>
      </div>

      {message ? <p className="access-handoff-message" role="status">{message}</p> : null}
      {!handoff && loading ? <div className="access-handoff-loading">{t("accessHandoff.loading")}</div> : null}
      {handoff && !ready ? (
        <div className="access-handoff-blocked">
          <ShieldCheck size={18} />
          <div>
            <strong>{t("accessHandoff.blockedTitle")}</strong>
            <span>{tx(t, "accessHandoff.blockerCount", { count: handoff.tokenEligibility.blockerCodes.length })}</span>
          </div>
        </div>
      ) : null}

      {handoff && ready ? (
        <>
          <div className="access-handoff-summary">
            <div>
              <span>{t("accessHandoff.allowedCapabilities")}</span>
              <strong>{handoff.allowedCapabilities.length}</strong>
            </div>
            <div>
              <span>{t("accessHandoff.blockedCapabilities")}</span>
              <strong>{handoff.blockedCapabilities.length}</strong>
            </div>
            <div>
              <span>{t("accessHandoff.subject")}</span>
              <strong>{handoff.scope.subjectSelector || t("text.notSpecified")}</strong>
            </div>
            <div>
              <span>{t("section.dataScope")}</span>
              <strong>{dataScopeText(handoff.dataScopes, t) || t("text.noDataScope")}</strong>
            </div>
          </div>

          <div className="access-handoff-capabilities">
            <div>
              <strong>{t("accessHandoff.availableResources")}</strong>
              <div className="access-handoff-chips">
                {handoff.allowedCapabilities.map((capability) => (
                  <span className="access-handoff-chip tone-success" key={capability.id}>
                    {capability.displayName || readableIdentifierLabel(capability.key)}
                  </span>
                ))}
              </div>
            </div>
            {handoff.blockedCapabilities.length > 0 ? (
              <div>
                <strong>{t("accessHandoff.unavailableResources")}</strong>
                <div className="access-handoff-chips">
                  {handoff.blockedCapabilities.map((capability) => (
                    <span className="access-handoff-chip tone-danger" key={capability.id}>
                      {capability.displayName || readableIdentifierLabel(capability.key)}
                    </span>
                  ))}
                </div>
              </div>
            ) : null}
          </div>

          {handoff.copyArtifacts ? (
            <div className="access-handoff-artifacts">
              <AccessHandoffArtifact
                copied={copied === "mcp"}
                label={t("accessHandoff.mcpConfig")}
                onCopy={() => void copyValue("mcp", handoff.copyArtifacts?.mcpClientConfig ?? "")}
                value={handoff.copyArtifacts.mcpClientConfig}
                t={t}
              />
              <AccessHandoffArtifact
                copied={copied === "prompt"}
                label={t("accessHandoff.promptTemplate")}
                onCopy={() => void copyValue("prompt", handoff.copyArtifacts?.promptTemplate ?? "")}
                value={handoff.copyArtifacts.promptTemplate}
                t={t}
              />
            </div>
          ) : null}

        </>
      ) : null}
      {handoff ? (
        <AccessHandoffTokenSection
          canCreate={ready}
          copied={copied === "token"}
          handoff={handoff}
          language={language}
          oneTimeToken={oneTimeToken}
          onClearOneTimeToken={controller.clearOneTimeToken}
          onCopyToken={(value) => void copyValue("token", value)}
          onCreateToken={() => void controller.createToken()}
          onRevokeToken={revokeToken}
          tokenAction={tokenAction}
          t={t}
        />
      ) : null}
    </section>
  );
}

function AccessHandoffTokenSection({
  canCreate,
  copied,
  handoff,
  language,
  oneTimeToken,
  onClearOneTimeToken,
  onCopyToken,
  onCreateToken,
  onRevokeToken,
  tokenAction,
  t
}: {
  canCreate: boolean;
  copied: boolean;
  handoff: AccessHandoff;
  language: Language;
  oneTimeToken: CreateAccessHandoffTokenResponse | null;
  onClearOneTimeToken: () => void;
  onCopyToken: (value: string) => void;
  onCreateToken: () => void;
  onRevokeToken: (id: string) => void;
  tokenAction: "" | "create" | "revoke";
  t: Translator;
}) {
  const activeToken = handoff.tokens.find((token) => token.status === "active");
  return (
    <section className="access-handoff-token-section">
      <div className="access-handoff-token-heading">
        <div>
          <span className="section-kicker">{t("accessHandoff.tokenKicker")}</span>
          <strong>{t("accessHandoff.tokenTitle")}</strong>
          <p>{canCreate ? t("accessHandoff.tokenDetail") : t("accessHandoff.tokenBlockedDetail")}</p>
        </div>
        {canCreate ? (
          <button
            className="primary-button"
            disabled={Boolean(activeToken) || tokenAction !== "" || !handoff.tokenEligibility.eligible}
            onClick={onCreateToken}
            type="button"
          >
            <KeyRound size={14} />
            {tokenAction === "create" ? t("accessHandoff.tokenCreating") : activeToken ? t("accessHandoff.tokenActive") : t("accessHandoff.tokenCreate")}
          </button>
        ) : null}
      </div>

      {oneTimeToken ? (
        <div className="access-handoff-one-time-token">
          <div>
            <strong>{t("accessHandoff.oneTimeToken")}</strong>
            <span>{t("accessHandoff.oneTimeTokenDetail")}</span>
          </div>
          <code translate="no">{oneTimeToken.key}</code>
          <div className="access-handoff-one-time-actions">
            <button className="secondary-button" onClick={() => onCopyToken(oneTimeToken.key)} type="button">
              <Copy size={14} />
              {copied ? t("action.copied") : t("action.copy")}
            </button>
            <button className="ghost-button" onClick={onClearOneTimeToken} type="button">
              <X size={14} />
              {t("action.dismiss")}
            </button>
          </div>
        </div>
      ) : null}

      <div className="access-handoff-token-list">
        {handoff.tokens.length === 0 ? <span>{t("accessHandoff.noTokens")}</span> : handoff.tokens.map((token) => (
          <div className="access-handoff-token-row" key={token.id}>
            <div>
              <strong translate="no">{token.prefix}…</strong>
              <span>{t(`accessHandoff.tokenStatus.${token.status}`)} · {formatDate(token.expiresAt, language)}</span>
            </div>
            {token.status === "active" ? (
              <button className="danger-button" disabled={tokenAction !== ""} onClick={() => onRevokeToken(token.id)} type="button">
                <Trash2 size={14} />
                {tokenAction === "revoke" ? t("accessHandoff.tokenRevoking") : t("accessHandoff.tokenRevoke")}
              </button>
            ) : null}
          </div>
        ))}
      </div>
    </section>
  );
}

function AccessHandoffArtifact({
  copied,
  label,
  onCopy,
  t,
  value
}: {
  copied: boolean;
  label: string;
  onCopy: () => void;
  t: Translator;
  value: string;
}) {
  return (
    <article>
      <div>
        <strong>{label}</strong>
        <button className="secondary-button" onClick={onCopy} type="button">
          <Copy size={14} />
          {copied ? t("action.copied") : t("action.copy")}
        </button>
      </div>
      <pre><code translate="no">{value}</code></pre>
    </article>
  );
}
