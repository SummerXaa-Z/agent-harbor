import type { ReactNode } from "react";

import type { MetricTone } from "../consoleMetrics";

export function Badge({ tone, children }: { tone: MetricTone; children: ReactNode }) {
  return <span className={`badge tone-${tone}`}>{children}</span>;
}

export function EmptyRow({ title, detail }: { title: string; detail: string }) {
  return (
    <div className="empty-row">
      <strong>{title}</strong>
      <span>{detail}</span>
    </div>
  );
}
