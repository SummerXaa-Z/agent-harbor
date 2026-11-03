import { useState } from "react";
import { Button } from "../../ui/Button";
import { Field } from "../../ui/Field";
import { Modal } from "../../ui/Modal";
import type { Capability } from "../../../types";
import type { useRedesignI18n } from "../../hooks/useRedesignI18n";

// Risk level and data domain are drafts local to the modal; the parent keys
// the component by capability id so drafts never leak across selections.
export function CapabilitiesDetailModal({
  capability,
  domainOptions,
  acting,
  t,
  onClose,
  onApprove,
  onSave
}: {
  capability: Capability;
  domainOptions: readonly string[];
  acting: boolean;
  t: ReturnType<typeof useRedesignI18n>["t"];
  onClose: () => void;
  onApprove: (capability: Capability) => void;
  onSave: (riskLevel: Capability["riskLevel"], dataDomain: string) => void;
}) {
  const [risk, setRisk] = useState<Capability["riskLevel"]>(capability.riskLevel);
  const [domain, setDomain] = useState(
    capability.dataDomains?.[0] ?? capability.dataScopes?.find((scope) => scope.dataDomain)?.dataDomain ?? ""
  );

  return (
    <Modal
      footer={
        <>
          <Button onClick={onClose} variant="ghost">{t("rd.common.close")}</Button>
          <Button disabled={acting} onClick={() => onSave(risk, domain)} variant="primary">{t("rd.cap.save")}</Button>
        </>
      }
      onClose={onClose}
      open
      title={capability.displayName}
    >
      <div className="stack">
        <div className="mono small muted">{capability.key}</div>
        {capability.description ? <p className="muted small">{capability.description}</p> : null}
        {capability.discoveryStatus === "pending_review" ? (
          <Button disabled={acting} onClick={() => onApprove(capability)} variant="success">
            {t("rd.cap.approve")}
          </Button>
        ) : null}
        <div className="form-grid">
          <Field htmlFor="cap-risk" label={t("rd.col.risk")}>
            <select
              className="select"
              id="cap-risk"
              onChange={(event) => setRisk(event.target.value as Capability["riskLevel"])}
              value={risk}
            >
              {(["low", "medium", "high", "critical"] as const).map((level) => (
                <option key={level} value={level}>{t(`rd.risk.${level}`)}</option>
              ))}
            </select>
          </Field>
          <Field htmlFor="cap-domain" label={t("rd.cap.domain")}>
            <select
              className="select"
              id="cap-domain"
              onChange={(event) => setDomain(event.target.value)}
              value={domain}
            >
              <option value="">{t("rd.cap.domainUnset")}</option>
              {domainOptions.map((option) => (
                <option key={option} value={option}>{option}</option>
              ))}
            </select>
          </Field>
        </div>
      </div>
    </Modal>
  );
}
