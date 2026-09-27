import type { ReactNode } from "react";
import { ToneIcon } from "./icons";

export type BannerTone = "info" | "success" | "warning" | "danger";

interface BannerProps {
  actions?: ReactNode;
  desc?: ReactNode;
  title: ReactNode;
  tone: BannerTone;
}

// One status banner per page: icon, title and description, then the page's
// single primary action plus secondary ones.
export function Banner({ actions, desc, title, tone }: BannerProps) {
  return (
    <div className={`banner banner-${tone}`} role={tone === "danger" ? "alert" : "status"}>
      <span className="b-ico">
        <ToneIcon size={18} tone={tone} />
      </span>
      <div className="b-main">
        <div className="b-title">{title}</div>
        {desc ? <div className="b-desc">{desc}</div> : null}
      </div>
      {actions ? <div className="b-actions">{actions}</div> : null}
    </div>
  );
}

export type NoticeTone = "info" | "warn" | "danger";

export function Notice({ children, tone = "info" }: { children: ReactNode; tone?: NoticeTone }) {
  const className = tone === "info" ? "notice" : `notice notice-${tone}`;
  return (
    <div className={className}>
      <ToneIcon size={15} tone={tone === "warn" ? "warning" : tone} />
      <div>{children}</div>
    </div>
  );
}
