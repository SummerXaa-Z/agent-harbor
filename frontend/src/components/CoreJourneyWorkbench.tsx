import {
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
  const runtimeEvidenceSummary = result
    ? `tools/list ${result.toolListStatus} · ${form.deniedTool} ${result.deniedStatus} · ${form.allowedTool} ${result.allowedStatus}`
    : t("text.selfCheckRuntimePending");
  return (
    <div className="core-journey">
      <p className="core-journey-intro">{t("text.coreJourneyIntro")}</p>
      <section className="core-journey-config">
        <header>
          <div>
            <strong>{t("section.selfCheckConfig")}</strong>
            <span>{t("text.selfCheckConfigDetail")}</span>
          </div>
          <div className="core-journey-score">
            <strong>{evaluation.completeCount}/{evaluation.totalCount}</strong>
            <span>{t("text.coreJourneyCompletion")}</span>
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
        <div className="core-journey-config-actions">
          <button className="primary-button" disabled={running || preflightChecking || !canRun} onClick={onRun} type="button">
            <Workflow size={14} />
            {running ? t("action.runningJourney") : t("action.runCoreJourney")}
          </button>
        </div>
      </section>

      <div className="core-journey-preflight">
        <div className="core-journey-preflight-header">
          <div>
            <strong>{t("section.preflight")}</strong>
            {preflightMessage ? <span>{preflightMessage}</span> : null}
          </div>
          <div className="core-journey-preflight-actions">
            <button className="secondary-button" disabled={running || preflightChecking} onClick={onRefreshPreflight} type="button">
              <RefreshCw size={14} />
              {preflightChecking ? t("action.checkingPreflight") : t("action.checkPreflight")}
            </button>
            <button className="secondary-button" disabled={running} onClick={onReset} type="button">
              <RefreshCw size={14} />
              {t("action.resetCoreJourney")}
            </button>
          </div>
        </div>
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
      </div>

      <div className="core-journey-runtime-summary" aria-label={t("section.selfCheckRuntime")}>
        <TechnicalId copyLabel={t("action.copy")} label={t("detail.runId")} value={config.runId} />
        <TechnicalId copyLabel={t("action.copy")} label={t("form.tenantId")} value={config.childTenantId} />
        <span className="core-journey-runtime-field">
          <span>{t("form.allowedTool")}</span>
          <code translate="no">{form.allowedTool}</code>
        </span>
        <span className="core-journey-runtime-field">
          <span>{t("form.deniedTool")}</span>
          <code translate="no">{form.deniedTool}</code>
        </span>
        <span className="core-journey-runtime-field is-wide">
          <span>{t("section.selfCheckRuntime")}</span>
          <strong>{runtimeEvidenceSummary}</strong>
        </span>
        {message ? <strong className="core-journey-message">{message}</strong> : null}
      </div>

      <div className="core-journey-steps">
        {evaluation.steps.map((step) => (
          <CoreJourneyStepRow key={step.key} step={step} t={t} />
        ))}
      </div>

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
  return (
    <article className={`core-journey-step status-${step.status}`}>
      <Badge tone={coreJourneyStatusTone(step.status)}>{coreJourneyStatusLabel(step.status, t)}</Badge>
      <div>
        <strong>{t(`journey.step.${step.key}`)}</strong>
        <span className="core-journey-step-detail" translate="no">{step.detail}</span>
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
