import { Check, X } from "lucide-react";
import { useRedesignI18n } from "../hooks/useRedesignI18n";

export type StepState = "done" | "current" | "blocked" | "skipped" | "pending";

export interface StepItem {
  key: string;
  label: string;
  state: StepState;
}

export function Stepper({ label, steps }: { label: string; steps: readonly StepItem[] }) {
  const { t } = useRedesignI18n();
  return (
    <ol aria-label={label} className="stepper">
      {steps.map((step, index) => (
        <li
          aria-current={step.state === "current" ? "step" : undefined}
          className={step.state === "pending" ? "step" : `step step-${step.state}`}
          key={step.key}
        >
          <span className="step-dot">
            {step.state === "done" ? <Check aria-hidden="true" size={15} /> : null}
            {step.state === "blocked" ? <X aria-hidden="true" size={15} /> : null}
            {step.state !== "done" && step.state !== "blocked" ? index + 1 : null}
          </span>
          <span className="step-label">{step.label}</span>
          <span className="step-state">{t(`rd.step.${step.state}`)}</span>
        </li>
      ))}
    </ol>
  );
}
