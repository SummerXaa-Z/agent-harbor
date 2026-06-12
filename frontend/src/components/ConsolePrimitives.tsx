import type { ReactNode } from "react";
import { ExternalLink, MoreHorizontal } from "lucide-react";

import type { Tone } from "../consolePresenters";

export function MetricCard({
  icon,
  label,
  value,
  detail,
  tone
}: {
  icon: ReactNode;
  label: string;
  value: string;
  detail: string;
  tone: Tone;
}) {
  return (
    <article className={`metric-card tone-${tone}`}>
      <div className="metric-icon">{icon}</div>
      <div>
        <span>{label}</span>
        <strong>{value}</strong>
        <small>{detail}</small>
      </div>
    </article>
  );
}

export function Panel({
  title,
  icon,
  action,
  className,
  children
}: {
  title: string;
  icon: ReactNode;
  action?: ReactNode;
  className?: string;
  children: ReactNode;
}) {
  return (
    <section className={`panel ${className ?? ""}`}>
      <header className="panel-header">
        <div>
          {icon}
          <h2>{title}</h2>
        </div>
        {action}
      </header>
      {children}
    </section>
  );
}

export function ActionDisclosurePanel({
  title,
  icon,
  className = "span-4",
  id,
  children
}: {
  title: string;
  icon: ReactNode;
  className?: string;
  id?: string;
  children: ReactNode;
}) {
  return (
    <details className={`action-disclosure-panel ${className}`} id={id}>
      <summary>
        <div>
          {icon}
          <strong>{title}</strong>
        </div>
      </summary>
      <div className="action-disclosure-body">
        {children}
      </div>
    </details>
  );
}

export function IconMore({ title = "More" }: { title?: string }) {
  return (
    <button aria-label={title} className="icon-button compact" title={title} type="button">
      <MoreHorizontal size={16} />
    </button>
  );
}

export function IconOpen({ title = "Open" }: { title?: string }) {
  return (
    <button aria-label={title} className="icon-button compact" title={title} type="button">
      <ExternalLink size={15} />
    </button>
  );
}
