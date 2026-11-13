import { traceResultKind, traceSource } from "../../model/auditTimeline";
import { Chip } from "../../ui/Chip";
import { KvList, type KvItem } from "../../ui/KvList";
import type { TraceEvent } from "../../../types";
import type { useRedesignI18n } from "../../hooks/useRedesignI18n";

// The decision explainer: the structured answer to "why was this call
// allowed or denied" above the raw record. Fields not recorded for a trace
// (legacy rows predate policy evaluation) simply stay absent.
export function TracePolicyDetail({ trace, t }: { trace: TraceEvent; t: ReturnType<typeof useRedesignI18n>["t"] }) {
  const items: KvItem[] = [];
  if (trace.source) {
    items.push({ key: "source", label: t("rd.audit.sourceLabel"), value: t(`rd.audit.source.${traceSource(trace)}`) });
  }
  if (trace.outcome) {
    items.push({ key: "outcome", label: t("rd.audit.outcomeLabel"), value: t(`rd.audit.result.${traceResultKind(trace)}`) });
  }
  if (trace.toolName) {
    items.push({ key: "tool", label: t("rd.audit.toolName"), value: <span className="mono small">{trace.toolName}</span> });
  }
  if (trace.reason) {
    items.push({ key: "reason", label: t("rd.audit.policyReason"), value: trace.reason });
  }
  if (trace.ruleRef) {
    items.push({ key: "rule", label: t("rd.audit.policyRule"), value: <span className="mono small">{trace.ruleRef}</span> });
  }
  if (trace.argsDigest) {
    items.push({
      key: "args",
      label: t("rd.audit.policyArgs"),
      value: (
        <span className="mono small">
          {trace.argsDigest}
          <span className="muted"> · {t("rd.audit.policyArgsNote")}</span>
        </span>
      ),
    });
  }
  if (items.length === 0) {
    return null;
  }
  return (
    <div className="stack">
      <div className="page-head-main">
        <h3>{t("rd.audit.policySection")}</h3>
      </div>
      <KvList items={items} />
      {trace.riskFlags && trace.riskFlags.length > 0 ? (
        <div className="apr-button-row">
          {trace.riskFlags.map((flag) => (
            <Chip key={flag} tone="danger">{flag}</Chip>
          ))}
        </div>
      ) : null}
    </div>
  );
}
