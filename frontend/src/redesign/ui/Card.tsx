import type { ReactNode } from "react";

interface CardProps {
  children: ReactNode;
  // Tables and lists that bring their own padding sit flush with the border.
  flush?: boolean;
  right?: ReactNode;
  sub?: ReactNode;
  title?: ReactNode;
}

export function Card({ children, flush = false, right, sub, title }: CardProps) {
  return (
    <section className="card">
      {title ? <CardHeader right={right} sub={sub} title={title} /> : null}
      {flush ? children : <div className="card-b">{children}</div>}
    </section>
  );
}

export function CardHeader({ right, sub, title }: { right?: ReactNode; sub?: ReactNode; title: ReactNode }) {
  return (
    <div className="card-h">
      <h3>{title}</h3>
      {sub ? <span className="card-sub">{sub}</span> : null}
      {right ? <div className="card-right">{right}</div> : null}
    </div>
  );
}
