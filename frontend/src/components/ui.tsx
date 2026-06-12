import type { ReactNode } from "react";
import { Inbox } from "lucide-react";

import type { MetricTone } from "../consoleMetrics";

export function Badge({ tone, children }: { tone: MetricTone; children: ReactNode }) {
  return <span className={`badge tone-${tone}`}>{children}</span>;
}

export function EmptyRow({ title, detail }: { title: string; detail: string }) {
  return (
    <div className="empty-row">
      <Inbox aria-hidden="true" className="empty-row-icon" size={18} />
      <div className="empty-row-copy">
        <strong>{title}</strong>
        <span>{detail}</span>
      </div>
    </div>
  );
}
