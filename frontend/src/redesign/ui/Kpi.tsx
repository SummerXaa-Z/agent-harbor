import type { ReactNode } from "react";

export function Kpi({ hint, icon, label, value }: { hint?: ReactNode; icon: ReactNode; label: ReactNode; value: ReactNode }) {
  return (
    <div className="card kpi">
      <div className="kpi-top">
        <span className="kpi-ico">{icon}</span>
      </div>
      <div className="kpi-num">{value}</div>
      <div className="kpi-label">{label}</div>
      {hint ? <div className="hint">{hint}</div> : null}
    </div>
  );
}
