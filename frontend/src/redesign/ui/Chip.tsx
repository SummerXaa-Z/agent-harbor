import type { ReactNode } from "react";
import { ToneIcon, type Tone } from "./icons";

export function Chip({ children, icon = true, tone }: { children: ReactNode; icon?: boolean; tone: Tone }) {
  return (
    <span className={`chip chip-${tone}`}>
      {icon ? <ToneIcon size={13} tone={tone} /> : null}
      {children}
    </span>
  );
}

export function TagOutline({ children }: { children: ReactNode }) {
  return <span className="tag-outline">{children}</span>;
}

export type RiskLevel = "low" | "mid" | "high";

// The badge shows one short glyph, so the full label rides along as the
// accessible name and tooltip.
export function Risk({ glyph, label, level }: { glyph: string; label: string; level: RiskLevel }) {
  return (
    <span aria-label={label} className={`risk risk-${level}`} role="img" title={label}>
      {glyph}
    </span>
  );
}
