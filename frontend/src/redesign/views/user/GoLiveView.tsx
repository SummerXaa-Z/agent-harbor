import { Copy, FileDown, KeyRound, Play, RefreshCw } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { apiBase, fetchPermissionPackageProductionReadiness } from "../../../api";
import { useAccessHandoffController } from "../../../hooks/useAccessHandoffController";
import { useConnectionDiagnostics } from "../../../hooks/useConnectionDiagnostics";
import { tx } from "../../../localizedMessages";
import type { PermissionPackageProductionReadiness } from "../../../permissionPackages";
import { useAccessContext } from "../../hooks/useAccessContext";
import { usePermissionCatalog } from "../../hooks/usePermissionCatalog";
import { usePermissionChangeFlow } from "../../hooks/usePermissionChangeFlow";
import { useRuntimeValidation } from "../../hooks/useRuntimeValidation";
import { useRedesignI18n } from "../../hooks/useRedesignI18n";
import { accessContextComplete, readinessFilterFromContext } from "../../model/accessContext";
import type { RuntimeValidationBlocker } from "../../model/runtimeValidation";
import { apiErrorPresentation } from "../../model/apiErrorCategory";
import { defaultTokenTtl, goLiveLegs, handoffShellSnippet, readinessCheckCount, tokenTtlOptions } from "../../model/goLive";
import { maskSecret } from "../../model/secretMask";
import { userHash, viewLabelKey } from "../../router";
import { DataStatusBanner, DataStatusChip } from "../../shell/DataStatusBanner";
import { Banner, Notice } from "../../ui/Banner";
import { Button } from "../../ui/Button";
import { Card } from "../../ui/Card";
import { Chip, TagOutline } from "../../ui/Chip";
import { CodeBlock } from "../../ui/CodeBlock";
import { Field } from "../../ui/Field";
import { KvList } from "../../ui/KvList";
import { Modal } from "../../ui/Modal";
import { EmptyState, LoadingState } from "../../ui/StateViews";
import { Table } from "../../ui/Table";
import { useToast } from "../../ui/Toast";
import { AccessContextSwitcher } from "./AccessContextSwitcher";
import { dataScopeLabel } from "./AskView";
import type { UserViewProps } from "./userViewProps";

const legTone = { attention: "warning", blocked: "danger", pending: "neutral", ready: "success" } as const;
const tokenStatusTone = { active: "success", expired: "neutral", revoked: "danger" } as const;

export function GoLiveView({ data, onRetry, params }: UserViewProps) {
  const { language, t } = useRedesignI18n();
  const showToast = useToast();
  const live = Boolean(data.data?.loadedFromApi);
  const consoleData = data.data;
  const accessContext = useAccessContext(data, params);
  const catalog = usePermissionCatalog(live);
  const context = accessContext.context;
  const flow = usePermissionChangeFlow({ context, live });
  const complete = accessContextComplete(context);
  const filter = useMemo(() => readinessFilterFromContext(context, flow.approval?.id), [context, flow.approval?.id]);
  const [readiness, setReadiness] = useState<PermissionPackageProductionReadiness | null>(null);
  const [readinessError, setReadinessError] = useState<unknown>(null);
  const [readinessLoading, setReadinessLoading] = useState(false);
  const [readinessKey, setReadinessKey] = useState(0);
  const target = consoleData?.agents.find((agent) => agent.id === context.targetId);
  const mcpEndpoint = typeof target?.channelConfig?.endpoint === "string" ? target.channelConfig.endpoint : "";
  const diagnostics = useConnectionDiagnostics({
    liveDataLoaded: live,
    loadError: data.failed ? apiErrorPresentation(t, language, data.error, "error.consoleDataUnavailable").detail : "",
    mcpEndpoint,
  });
  const handoff = useAccessHandoffController({
    adminKey: "",
    enabled: live && complete,
    filter,
    language,
    refreshKey: String(readinessKey),
    t,
  });
  const validation = useRuntimeValidation({ context, live, preview: flow.preview });
  const [ttl, setTtl] = useState(defaultTokenTtl(null));
  const [copyOpen, setCopyOpen] = useState(false);
  const [revokeId, setRevokeId] = useState("");
  const plaintextRef = useRef("");
  const diagnosticsStarted = useRef(false);

  useEffect(() => {
    if (!live || diagnosticsStarted.current || diagnostics.checking || diagnostics.status !== null) return;
    diagnosticsStarted.current = true;
    void diagnostics.run();
  }, [diagnostics.checking, diagnostics.run, diagnostics.status, live]);

  useEffect(() => {
    if (!live || !complete) {
      setReadiness(null);
      return;
    }
    const controller = new AbortController();
    setReadinessLoading(true);
    fetchPermissionPackageProductionReadiness(filter, "", controller.signal)
      .then((next) => {
        setReadiness(next);
        setReadinessError(null);
      })
      .catch((error: unknown) => {
        if (controller.signal.aborted) return;
        setReadinessError(error);
        setReadiness(null);
      })
      .finally(() => {
        if (!controller.signal.aborted) setReadinessLoading(false);
      });
    return () => controller.abort();
  }, [complete, filter, live, readinessKey]);

  const ttlChoices = tokenTtlOptions(handoff.handoff);
  useEffect(() => {
    setTtl(defaultTokenTtl(handoff.handoff));
  }, [handoff.handoff]);

  const legs = goLiveLegs({
    approval: flow.approval,
    connectionStatus: diagnostics.status,
    liveDataAvailable: live,
    preview: flow.preview,
    readiness,
  });
  const checks = readinessCheckCount(readiness, flow.preview);
  const blockedCapabilityCount = flow.preview?.draft.blockedCapabilities.length ?? 0;
  // The runtime-evidence leg: allowed record always required, denied record
  // only when the draft actually blocks something (backend "not applicable").
  const runtimePending = readiness
    ? !(readiness.summary.hasAllowedTrace && (readiness.summary.hasDeniedTrace || blockedCapabilityCount === 0))
    : Boolean(flow.preview && !flow.preview.summary.runtimeEvidenceReady);
  const showValidationCard = live && complete && flow.preview !== null && (runtimePending || validation.running);
  const agents = consoleData?.agents ?? [];
  const shell = handoff.handoff
    ? handoffShellSnippet({
        apiBase,
        comments: { call: t("rd.golive.shellCallComment"), token: t("rd.golive.shellTokenComment") },
        handoff: handoff.handoff,
      })
    : "";

  function recheck() {
    setReadinessKey((key) => key + 1);
    flow.refresh();
    void diagnostics.run();
    void handoff.refresh();
  }

  async function runValidation() {
    if (await validation.run()) recheck();
  }

  const validationBlockerKeys: Record<RuntimeValidationBlocker, string> = {
    requiresAllowedCapability: "message.aiAdminRuntimeValidationNoAllowedCapability",
    requiresApplication: "message.aiAdminRuntimeValidationRequiresApplication",
    requiresLiveApi: "message.fallbackDataModeActionBlocked",
    requiresSubject: "rd.golive.validationRequiresSubject",
  };

  async function copyConfig() {
    const raw = handoff.handoff?.copyArtifacts?.mcpClientConfig ?? "";
    if (!raw) {
      showToast(t("rd.golive.copyConfigMissing"), "warning");
      return;
    }
    try {
      await navigator.clipboard.writeText(raw);
      showToast(t("rd.golive.copyConfigDone"));
    } catch {
      showToast(t("rd.common.copyFailed"), "danger");
    }
  }

  async function issueToken() {
    const issued = await handoff.createToken(ttl);
    if (!issued) return;
    plaintextRef.current = issued["key"];
    setCopyOpen(true);
  }

  function closeCopyModal() {
    plaintextRef.current = "";
    handoff.clearOneTimeToken();
    setCopyOpen(false);
  }

  async function copyIssuedSecret() {
    const secret = plaintextRef.current;
    if (!secret) {
      showToast(t("rd.golive.tokenCopyFailed"), "danger");
      return;
    }
    try {
      await navigator.clipboard.writeText(secret);
      showToast(t("rd.golive.tokenCopied"));
      closeCopyModal();
    } catch {
      showToast(t("rd.golive.tokenCopyFailed"), "danger");
    }
  }

  async function confirmRevoke() {
    if (!revokeId) return;
    await handoff.revokeToken(revokeId);
    setRevokeId("");
  }

  return (
    <>
      <div className="page-head">
        <div className="page-head-main">
          <h1>{t(viewLabelKey("golive"))}</h1>
          <p>{t("rd.page.golive.desc")}</p>
        </div>
        <div className="page-head-aside">
          <DataStatusChip status={data.status} />
        </div>
      </div>
      <DataStatusBanner data={data} onRetry={onRetry} />
      {!consoleData ? (
        <LoadingState label={t("rd.data.loading")} />
      ) : (
        <div className="stack">
          <Card right={<AccessContextSwitcher accessContext={accessContext} agents={agents} templates={catalog.templates} />} title={t("rd.context.label")}>
            {!complete ? <Notice>{t("rd.golive.noScopeDesc")}</Notice> : null}
          </Card>

          {legs ? (
            <Banner
              actions={
                <>
                  <Button
                    disabled={flow.busy !== "" || !live}
                    icon={<FileDown aria-hidden="true" size={15} />}
                    // The report is always fetched fresh; the re-check first
                    // keeps the on-page snapshot from lagging what was just
                    // exported (round 4, #32).
                    onClick={() => {
                      recheck();
                      void flow.exportReport();
                    }}
                  >
                    {t("action.exportAcceptanceReport")}
                  </Button>
                  <Button icon={<RefreshCw aria-hidden="true" size={15} />} onClick={recheck} variant="ghost">
                    {t("rd.golive.recheck")}
                  </Button>
                  <Button href={userHash("home")} variant="ghost">
                    {t("rd.golive.viewRequests")}
                  </Button>
                </>
              }
              desc={tx(t, "rd.golive.legsCount", { done: legs.readyCount, total: legs.totalCount })}
              title={t(legs.headlineKey)}
              tone={legs.status === "ready" ? "success" : legs.status === "blocked" ? "danger" : "warning"}
            />
          ) : (
            <Banner desc={t("rd.golive.noScopeDesc")} title={t("rd.golive.noScope")} tone="info" />
          )}

          {readinessError ? <Notice tone="danger">{apiErrorPresentation(t, language, readinessError, "error.checkProductionReadiness").detail}</Notice> : null}
          {handoff.message ? <Notice>{handoff.message}</Notice> : null}

          <Card sub={legs ? tx(t, "rd.golive.legsCount", { done: legs.readyCount, total: legs.totalCount }) : undefined} title={t("rd.golive.legsTitle")}>
            {readinessLoading && !legs ? (
              <LoadingState label={t("rd.common.loading")} />
            ) : legs ? (
              <div className="grid grid-4">
                {legs.checkRows.map((row) => (
                  <div className="leg-card" key={row.key}>
                    <Chip tone={legTone[row.status]}>{t(`productionAcceptance.status.${row.status}`)}</Chip>
                    <div className="leg-title">{t(row.labelKey)}</div>
                    <p className="leg-desc">{row.detail ?? (row.detailKey ? t(row.detailKey) : "")}</p>
                    {checks ? (
                      <div className="leg-checks">{tx(t, "rd.golive.readinessCount", { ready: checks.ready, total: checks.total })}</div>
                    ) : null}
                  </div>
                ))}
              </div>
            ) : (
              <EmptyState desc={t("rd.golive.noScopeDesc")} title={t("rd.golive.legsTitle")} />
            )}
          </Card>

          {showValidationCard ? (
            <Card sub={t("rd.golive.validationDesc")} title={t("rd.golive.validationTitle")}>
              {validation.plan ? (
                <div className="stack">
                  <KvList
                    items={[
                      { key: "subject", label: t("rd.apply.subject"), value: <span className="mono small">{validation.plan.subjectId}</span> },
                      { key: "allowed", label: t("rd.golive.validationAllowedTool"), value: <span className="mono small">{validation.plan.allowedCapabilityKey}</span> },
                      {
                        key: "blocked",
                        label: t("rd.golive.validationBlockedTool"),
                        value: validation.plan.blockedCapabilityKey
                          ? <span className="mono small">{validation.plan.blockedCapabilityKey}</span>
                          : t("rd.golive.validationNoBlocked"),
                      },
                    ]}
                  />
                  <div className="form-actions">
                    <Button disabled={validation.running} icon={<Play aria-hidden="true" size={15} />} onClick={() => void runValidation()}>
                      {validation.running ? t("rd.golive.validationRunning") : t("rd.golive.validationRun")}
                    </Button>
                  </div>
                  <p className="hint">{t("rd.golive.validationPlanNote")}</p>
                </div>
              ) : (
                <div className="stack">
                  <Notice tone="warn">
                    {validation.blockers.map((blocker) => t(validationBlockerKeys[blocker])).join(" ")}
                  </Notice>
                  <p className="hint">{t("rd.golive.validationPlanNote")}</p>
                </div>
              )}
            </Card>
          ) : null}

          <Card title={t("rd.golive.handoffTitle")}>
            {!handoff.handoff ? (
              handoff.loading ? (
                <LoadingState label={t("accessHandoff.loading")} />
              ) : (
                <EmptyState desc={t("rd.golive.noScopeDesc")} title={t("rd.golive.handoffTitle")} />
              )
            ) : (
              <div className="stack">
                <div>
                  <div className="field-label">{t("rd.golive.boundaryTitle")}</div>
                  <div className="chip-row">
                    {handoff.handoff.allowedCapabilities.map((capability) => (
                      <Chip key={capability.id} tone="success">
                        {capability.displayName || capability.key}
                      </Chip>
                    ))}
                    {handoff.handoff.allowedCapabilities.length === 0 ? <span className="muted small">{t("rd.common.none")}</span> : null}
                  </div>
                  {handoff.handoff.blockedCapabilities.length > 0 ? (
                    <div className="chip-row">
                      {handoff.handoff.blockedCapabilities.map((capability) => (
                        <TagOutline key={capability.id}>{capability.displayName || capability.key}</TagOutline>
                      ))}
                    </div>
                  ) : null}
                  <p className="hint">
                    {handoff.handoff.dataScopes.map(dataScopeLabel).filter(Boolean).join("; ") || t("rd.common.none")}
                  </p>
                </div>
                <div>
                  <div className="field-label">{t("rd.golive.shellTitle")}</div>
                  <CodeBlock code={shell} label={t("rd.golive.shellTitle")} />
                </div>
                <div className="form-actions">
                  <Button icon={<Copy aria-hidden="true" size={15} />} onClick={() => void copyConfig()} variant="ghost">
                    {t("rd.golive.copyConfig")}
                  </Button>
                  <Field htmlFor="rd-golive-ttl" label={t("rd.golive.ttl")}>
                    <select className="select" id="rd-golive-ttl" onChange={(event) => setTtl(Number(event.target.value))} value={ttl}>
                      {ttlChoices.map((seconds) => (
                        <option key={seconds} value={seconds}>
                          {t(`rd.golive.ttl.${seconds}`)}
                        </option>
                      ))}
                    </select>
                  </Field>
                  <Button
                    disabled={!handoff.handoff.tokenEligibility.eligible || handoff.tokenAction !== ""}
                    icon={<KeyRound aria-hidden="true" size={15} />}
                    onClick={() => void issueToken()}
                  >
                    {handoff.tokenAction === "create" ? t("accessHandoff.tokenCreating") : t("rd.golive.createToken")}
                  </Button>
                </div>
              </div>
            )}
          </Card>

          <Card flush title={t("rd.golive.tokensTitle")}>
            <Table
              caption={t("rd.golive.tokensTitle")}
              renderCell={(row, column) => {
                switch (column) {
                  case "token":
                    return <span className="mono small">{maskSecret(row.prefix)}</span>;
                  case "status":
                    return <Chip tone={tokenStatusTone[row.status]}>{t(`rd.tokenStatus.${row.status}`)}</Chip>;
                  case "expiresAt":
                    return row.expiresAt;
                  case "subject":
                    return row.subjectSelector ? <span className="mono small">{row.subjectSelector}</span> : <span className="muted">{t("rd.common.none")}</span>;
                  case "actions":
                    return row.status === "active" ? (
                      <Button onClick={() => setRevokeId(row.id)} size="sm" variant="danger-ghost">
                        {t("rd.golive.revoke")}
                      </Button>
                    ) : (
                      <span className="muted">{t("rd.common.none")}</span>
                    );
                }
              }}
              rowKey={(row) => row.id}
              rows={handoff.handoff?.tokens ?? []}
              tableId="userTokens"
            />
          </Card>
        </div>
      )}

      <Modal
        footer={
          <Button onClick={() => void copyIssuedSecret()}>
            {t("rd.golive.copyToken")}
          </Button>
        }
        onClose={closeCopyModal}
        open={copyOpen}
        size="narrow"
        title={t("rd.golive.tokenModalTitle")}
      >
        <p>{t("rd.golive.tokenModalDesc")}</p>
      </Modal>

      <Modal
        footer={
          <>
            <Button onClick={() => setRevokeId("")} variant="ghost">
              {t("rd.common.close")}
            </Button>
            <Button onClick={() => void confirmRevoke()} variant="danger-ghost">
              {t("rd.golive.revoke")}
            </Button>
          </>
        }
        onClose={() => setRevokeId("")}
        open={Boolean(revokeId)}
        size="narrow"
        title={t("rd.golive.revokeConfirm")}
      >
        <p>{t("rd.golive.revokeConfirmDesc")}</p>
      </Modal>
    </>
  );
}
