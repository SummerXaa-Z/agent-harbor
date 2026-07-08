import {
  ChevronRight,
  DatabaseZap,
  FileSearch,
  LockKeyhole,
  RefreshCw,
  Workflow
} from "lucide-react";

import type { Translator, Tone } from "../consolePresenters";
import type { NavKey } from "../consoleNavigation";
import type {
  CoreJourneyConfig,
  CoreJourneyEvaluation,
  CoreJourneyForm,
  CoreJourneyStep,
  CoreJourneyStepStatus
} from "../coreJourney";
import { coreJourneyStepDetailLabel } from "../coreJourneyPresentation";
import {
  coreJourneyPreflightCanRun,
  coreJourneyPreflightRows,
  type CoreJourneyPreflightState,
  type CoreJourneyPreflightStatus
} from "../coreJourneyPreflight";
import { TechnicalId } from "./TechnicalId";
import { Badge } from "./ui";

interface CoreJourneyRunResult {
  allowedStatus: number;
  callerId: string;
  deniedStatus: number;
  targetId: string;
  toolListStatus: number;
}

export function CoreJourneyWorkbench({
  config,
  evaluation,
  form,
  message,
  onChange,
  onOpen,
  onRefreshPreflight,
  onReset,
  onRun,
  preflight,
  preflightChecking,
  preflightMessage,
  result,
  running,
  t
}: {
  config: CoreJourneyConfig;
  evaluation: CoreJourneyEvaluation;
  form: CoreJourneyForm;
  message: string;
  onChange: (form: CoreJourneyForm) => void;
  onOpen: (key: NavKey) => void;
  onRefreshPreflight: () => void;
  onReset: () => void;
  onRun: () => void;
  preflight: CoreJourneyPreflightState;
  preflightChecking: boolean;
  preflightMessage: string;
  result: CoreJourneyRunResult | null;
  running: boolean;
  t: Translator;
}) {
  const canRun = coreJourneyPreflightCanRun(preflight);
  const healthTone: Tone = canRun ? "success" : "warning";
  const healthStatusLabel = canRun ? t("status.systemHealthReady") : t("status.systemHealthNeedsCheck");
  const healthTitle = canRun ? t("text.systemHealthReadyTitle") : t("text.systemHealthNeedsCheckTitle");
  const healthDetail = canRun ? t("text.systemHealthReadyDetail") : t("text.systemHealthNeedsCheckDetail");
  const runtimeRecordSummary = result
    ? `tools/list ${result.toolListStatus} · ${form.deniedTool} ${result.deniedStatus} · ${form.allowedTool} ${result.allowedStatus}`
    : t("text.selfCheckRuntimePending");
  return (
    <div className="core-journey">
      <section className="core-journey-health">
        <div className="core-journey-health-summary">
          <span>{t("section.systemHealthStatus")}</span>
          <div>
            <strong>{healthTitle}</strong>
            <Badge tone={healthTone}>{healthStatusLabel}</Badge>
          </div>
          <p>{healthDetail}</p>
          {message ? <strong className="core-journey-message">{message}</strong> : null}
        </div>
        <div className="core-journey-health-status">
          <div className="core-journey-score">
            <strong>{evaluation.completeCount}/{evaluation.totalCount}</strong>
            <span>{t("text.coreJourneyCompletion")}</span>
          </div>
          <div className="core-journey-health-actions">
            <button className="secondary-button" disabled={running || preflightChecking} onClick={onRefreshPreflight} type="button">
              <RefreshCw size={14} />
              {preflightChecking ? t("action.checkingPreflight") : t("action.checkPreflight")}
            </button>
            <button className="secondary-button" disabled={running} onClick={onReset} type="button">
              <RefreshCw size={14} />
              {t("action.resetCoreJourney")}
            </button>
          </div>
          <button className="primary-button" disabled={running || preflightChecking || !canRun} onClick={onRun} type="button">
            <Workflow size={14} />
            {running ? t("action.runningJourney") : t("action.runCoreJourney")}
          </button>
        </div>
      </section>

      <section className="core-journey-task">
        <header>
          <div>
            <strong>{t("section.selfCheckTask")}</strong>
            <span>{t("text.coreJourneyIntro")}</span>
            {preflightMessage ? <span>{preflightMessage}</span> : null}
          </div>
        </header>
        <section className="core-journey-preflight" aria-label={t("section.preflight")}>
          <div className="core-journey-preflight-grid">
            {coreJourneyPreflightRows(preflight).map((row) => (
              <article className={`core-journey-preflight-row status-${row.status}`} key={row.key}>
                <Badge tone={preflightTone(row.status)}>{preflightStatusLabel(row.status, t)}</Badge>
                <div>
                  <strong>{t(row.titleKey)}</strong>
                  <span>{t(row.detailKey)}</span>
                </div>
              </article>
            ))}
          </div>
        </section>

        <div className="core-journey-steps">
          {evaluation.steps.map((step) => (
            <CoreJourneyStepRow key={step.key} step={step} t={t} />
          ))}
        </div>
      </section>

      <details className="core-journey-advanced">
        <summary>
          <div>
            <strong>{t("section.selfCheckAdvanced")}</strong>
            <span>{t("text.selfCheckAdvancedDetail")}</span>
          </div>
          <span className="core-journey-disclosure-action">
            {t("action.viewDetails")}
            <ChevronRight size={15} aria-hidden="true" />
          </span>
        </summary>
        <section className="core-journey-config">
          <header>
            <div>
              <strong>{t("section.selfCheckConfig")}</strong>
              <span>{t("text.selfCheckConfigDetail")}</span>
            </div>
          </header>
          <div className="core-journey-config-grid">
            <label>
              {t("form.workspace")}
              <input
                disabled={running}
                value={form.workspaceId}
                onChange={(event) => onChange({ ...form, workspaceId: event.target.value })}
              />
            </label>
            <label>
              {t("form.endpoint")}
              <input
                disabled={running}
                value={form.mcpEndpoint}
                onChange={(event) => onChange({ ...form, mcpEndpoint: event.target.value })}
              />
            </label>
            <label>
              {t("form.allowedTool")}
              <input
                disabled={running}
                value={form.allowedTool}
                onChange={(event) => onChange({ ...form, allowedTool: event.target.value })}
              />
            </label>
            <label>
              {t("form.deniedTool")}
              <input
                disabled={running}
                value={form.deniedTool}
                onChange={(event) => onChange({ ...form, deniedTool: event.target.value })}
              />
            </label>
          </div>
        </section>
      </details>

      <details className="core-journey-runtime-summary" aria-label={t("section.selfCheckRuntimeDetail")}>
        <summary>
          <div>
            <strong>{t("section.selfCheckRuntimeDetail")}</strong>
            <span>{runtimeRecordSummary}</span>
          </div>
          <span className="core-journey-disclosure-action">
            {t("action.viewDetails")}
            <ChevronRight size={15} aria-hidden="true" />
          </span>
        </summary>
        <div className="core-journey-runtime-cards">
          <span className="core-journey-runtime-card">
            <span>{t("section.selfCheckRuntimeScope")}</span>
            <strong>{t("text.selfCheckRuntimeScopeDetail")}</strong>
            <code translate="no">{form.mcpEndpoint}</code>
          </span>
          <span className="core-journey-runtime-card">
            <span>{t("section.selfCheckRuntimeDecision")}</span>
            <strong>{t("text.selfCheckRuntimeDecisionDetail")}</strong>
            <code translate="no">{form.allowedTool} / {form.deniedTool}</code>
          </span>
        </div>
        <div className="core-journey-runtime-diagnostics">
          <strong>{t("section.selfCheckDiagnosticIdentifiers")}</strong>
          <div className="core-journey-runtime-grid">
            <TechnicalId copyLabel={t("action.copy")} label={t("detail.runId")} value={config.runId} />
            <TechnicalId copyLabel={t("action.copy")} label={t("form.tenantId")} value={config.childTenantId} />
          </div>
        </div>
      </details>

      <div className="core-journey-actions">
        <button className="secondary-button" onClick={() => onOpen("access")} type="button">
          <LockKeyhole size={14} />
          {t("action.openAccess")}
        </button>
        <button className="secondary-button" onClick={() => onOpen("capabilities")} type="button">
          <DatabaseZap size={14} />
          {t("action.openCapabilities")}
        </button>
        <button className="secondary-button" onClick={() => onOpen("traces")} type="button">
          <FileSearch size={14} />
          {t("action.openTraces")}
        </button>
      </div>
    </div>
  );
}

function CoreJourneyStepRow({ step, t }: { step: CoreJourneyStep; t: Translator }) {
  const detail = coreJourneyStepDetailLabel(step, t);
  return (
    <article className={`core-journey-step status-${step.status}`}>
      <Badge tone={coreJourneyStatusTone(step.status)}>{coreJourneyStatusLabel(step.status, t)}</Badge>
      <div>
        <strong>{t(`journey.step.${step.key}`)}</strong>
        <span className="core-journey-step-detail" title={step.detail}>{detail}</span>
      </div>
      <code className="core-journey-step-metric">{step.metric}</code>
    </article>
  );
}

function coreJourneyStatusTone(status: CoreJourneyStepStatus): Tone {
  if (status === "complete") return "success";
  if (status === "partial") return "warning";
  return "neutral";
}

function coreJourneyStatusLabel(status: CoreJourneyStepStatus, t: Translator) {
  if (status === "complete") return t("status.stepComplete");
  if (status === "partial") return t("status.stepPartial");
  return t("status.stepMissing");
}

function preflightTone(status: CoreJourneyPreflightStatus): Tone {
  if (status === "ok") return "success";
  if (status === "warning") return "warning";
  if (status === "error") return "danger";
  return "neutral";
}

function preflightStatusLabel(status: CoreJourneyPreflightStatus, t: Translator) {
  if (status === "ok") return t("status.preflightOk");
  if (status === "warning") return t("status.preflightWarning");
  if (status === "error") return t("status.preflightError");
  return t("status.preflightPending");
}
