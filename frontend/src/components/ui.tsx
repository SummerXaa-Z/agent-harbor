import type { ReactNode } from "react";
import { Inbox } from "lucide-react";

import type { MetricTone } from "../consoleMetrics";

export function Badge({ tone, children }: { tone: MetricTone; children: ReactNode }) {
  return <span className={`badge tone-${tone}`}>{children}</span>;
}

interface EmptyRowProps {
  actionHash?: string;
  actionLabel?: string;
  detail: string;
  onAction?: () => void;
  title: string;
}

export function EmptyRow({ actionHash, actionLabel, detail, onAction, title }: EmptyRowProps) {
  const action = actionLabel
    ? actionHash
      ? (
        <a className="secondary-button empty-row-action" href={actionHash}>
          {actionLabel}
        </a>
      )
      : onAction
        ? (
          <button className="secondary-button empty-row-action" onClick={onAction} type="button">
            {actionLabel}
          </button>
        )
        : null
    : null;

  return (
    <div className="empty-row">
      <Inbox aria-hidden="true" className="empty-row-icon" size={18} />
      <div className="empty-row-copy">
        <strong>{title}</strong>
        <span>{detail}</span>
        {action}
      </div>
    </div>
  );
}
