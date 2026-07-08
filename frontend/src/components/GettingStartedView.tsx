import { ArrowRight, Check, Circle } from "lucide-react";

import type { GettingStartedStep } from "../gettingStarted";
import type { Translator } from "../consolePresenters";
import { Badge } from "./ui";

interface GettingStartedViewProps {
  setupDataAvailable: boolean;
  steps: GettingStartedStep[];
  t: Translator;
}

export function GettingStartedView({ setupDataAvailable, steps, t }: GettingStartedViewProps) {
  const currentIndex = steps.findIndex((step) => !step.done);
  const currentStepIndex = currentIndex === -1 ? steps.length - 1 : currentIndex;
  const focusStep = steps[currentStepIndex] ?? steps[0];
  const completedSteps = steps.filter((step) => step.done).length;
  const allStepsDone = steps.length > 0 && completedSteps === steps.length;
  const focusStatus = focusStep?.done ? "complete" : "current";
  const chainNodes = ["tenant", "agent", "capability", "grant", "runtime", "status"];

  function statusForStep(step: GettingStartedStep | undefined, index: number) {
    if (step?.done) return "complete";
    return index === currentStepIndex ? "current" : "pending";
  }

  return (
    <div className="span-12 getting-started">
      <header className="getting-started-header">
        <div>
          <span className="section-kicker">{t("gettingStarted.kicker")}</span>
          <h2>{t("gettingStarted.title")}</h2>
          <p>{t("gettingStarted.lead")}</p>
        </div>
        {!setupDataAvailable ? (
          <div className="getting-started-notice">
            <Badge tone="warning">{t("gettingStarted.sampleBadge")}</Badge>
            <span>{t("gettingStarted.sampleNotice")}</span>
          </div>
        ) : null}
      </header>

      {focusStep ? (
        <div className="getting-started-layout">
          <div className="getting-started-main">
            <section className={`getting-started-focus status-${focusStatus}`}>
              <span className="section-kicker">
                {allStepsDone ? t("gettingStarted.readyLabel") : t("gettingStarted.focusLabel")}
              </span>
              <div className="getting-started-focus-title">
                <span className="getting-started-focus-index" aria-hidden="true">
                  {focusStep.done ? <Check size={18} /> : currentStepIndex + 1}
                </span>
                <div>
                  <h3>{t(`gettingStarted.step.${focusStep.key}.title`)}</h3>
                  <p>{t(`gettingStarted.step.${focusStep.key}.detail`)}</p>
                </div>
              </div>
              <a className="primary-button" href={focusStep.targetHash}>
                <span>{t(`gettingStarted.step.${focusStep.key}.action`)}</span>
                <ArrowRight aria-hidden="true" size={14} />
              </a>
            </section>

            <section className="getting-started-step-list" aria-label={t("gettingStarted.stepsLabel")}>
              <ol className="getting-started-steps">
                {steps.map((step, index) => {
                  const status = statusForStep(step, index);
                  return (
                    <li className={`getting-started-step status-${status}`} key={step.key}>
                      <span className="getting-started-step-index" aria-hidden="true">
                        {step.done ? <Check size={15} /> : <Circle size={10} />}
                      </span>
                      <div className="getting-started-step-copy">
                        <strong>{t(`gettingStarted.step.${step.key}.title`)}</strong>
                        <span>{t(`gettingStarted.step.${step.key}.detail`)}</span>
                      </div>
                      <a className="secondary-button" href={step.targetHash}>
                        <span>{t(`gettingStarted.step.${step.key}.action`)}</span>
                        <ArrowRight aria-hidden="true" size={14} />
                      </a>
                    </li>
                  );
                })}
              </ol>
            </section>
          </div>

          <aside className="getting-started-summary" aria-label={t("gettingStarted.summaryLabel")}>
            <div className="getting-started-progress">
              <div>
                <span>{t("gettingStarted.progressLabel")}</span>
                <strong>
                  {completedSteps}/{steps.length}
                </strong>
              </div>
              <progress
                aria-label={t("gettingStarted.summaryLabel")}
                className="getting-started-progress-bar"
                max={steps.length}
                value={completedSteps}
              />
              <span>{t("gettingStarted.completedLabel")}</span>
            </div>

            <div className="getting-started-source">
              <span>{t("gettingStarted.dataSourceLabel")}</span>
              <strong>
                {setupDataAvailable ? t("gettingStarted.liveDataSource") : t("gettingStarted.sampleDataSource")}
              </strong>
            </div>

            <ol className="getting-started-chain" aria-label={t("gettingStarted.chainLabel")}>
              {chainNodes.map((node, index) => (
                <li className={`getting-started-chain-node status-${statusForStep(steps[index], index)}`} key={node}>
                  <span className="getting-started-chain-index" aria-hidden="true">
                    {steps[index]?.done ? <Check size={12} /> : index + 1}
                  </span>
                  <span>{t(`gettingStarted.chain.${node}`)}</span>
                </li>
              ))}
            </ol>
          </aside>
        </div>
      ) : null}
    </div>
  );
}
