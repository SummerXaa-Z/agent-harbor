import { ArrowRight, Check, Circle } from "lucide-react";

import type { GettingStartedStep } from "../gettingStarted";
import type { Translator } from "../consolePresenters";
import { Badge } from "./ui";

interface GettingStartedViewProps {
  liveDataAvailable: boolean;
  steps: GettingStartedStep[];
  t: Translator;
}

export function GettingStartedView({ liveDataAvailable, steps, t }: GettingStartedViewProps) {
  const currentIndex = steps.findIndex((step) => !step.done);
  const currentStepIndex = currentIndex === -1 ? steps.length - 1 : currentIndex;

  return (
    <div className="panel span-12 getting-started">
      <header className="getting-started-header">
        <div>
          <span className="section-kicker">{t("gettingStarted.kicker")}</span>
          <h2>{t("gettingStarted.title")}</h2>
          <p>{t("gettingStarted.lead")}</p>
        </div>
        {!liveDataAvailable ? (
          <div className="getting-started-notice">
            <Badge tone="warning">{t("gettingStarted.sampleBadge")}</Badge>
            <span>{t("gettingStarted.sampleNotice")}</span>
          </div>
        ) : null}
      </header>

      <ol className="getting-started-chain" aria-label={t("gettingStarted.chainLabel")}>
        {["tenant", "agent", "capability", "grant", "runtime", "evidence"].map((node) => (
          <li key={node}>{t(`gettingStarted.chain.${node}`)}</li>
        ))}
      </ol>

      <ol className="getting-started-steps">
        {steps.map((step, index) => {
          const status = step.done ? "complete" : index === currentStepIndex ? "current" : "pending";
          const showSampleBadge = !liveDataAvailable && step.key !== "connect-api" && step.done;
          return (
            <li className={`getting-started-step status-${status}`} key={step.key}>
              <span className="getting-started-step-index" aria-hidden="true">
                {step.done ? <Check size={15} /> : <Circle size={10} />}
              </span>
              <div className="getting-started-step-copy">
                <strong>{t(`gettingStarted.step.${step.key}.title`)}</strong>
                <span>{t(`gettingStarted.step.${step.key}.detail`)}</span>
                {showSampleBadge ? <Badge tone="warning">{t("gettingStarted.sampleBadge")}</Badge> : null}
              </div>
              <a className="secondary-button" href={step.targetHash}>
                <span>{t(`gettingStarted.step.${step.key}.action`)}</span>
                <ArrowRight aria-hidden="true" size={14} />
              </a>
            </li>
          );
        })}
      </ol>
    </div>
  );
}
