import { Button } from "../../ui/Button";
import { Card } from "../../ui/Card";
import { Chip, TagOutline } from "../../ui/Chip";
import { EmptyState } from "../../ui/StateViews";
import { Table } from "../../ui/Table";
import type { Capability } from "../../../types";
import type { useRedesignI18n } from "../../hooks/useRedesignI18n";

type CapabilityColumnKey = "capability" | "type" | "risk" | "dataScope" | "status" | "actions";

const capabilityStatusTone: Record<Capability["discoveryStatus"], "success" | "warning" | "neutral"> = {
  approved: "success",
  deprecated: "neutral",
  pending_review: "warning",
  removed: "neutral"
};

const capabilityStatusKey: Record<Capability["discoveryStatus"], string> = {
  approved: "status.capabilityApproved",
  deprecated: "status.capabilityDeprecated",
  pending_review: "status.capabilityPendingReview",
  removed: "status.capabilityRemoved"
};

export function CapabilitiesTable({
  capabilities,
  acting,
  t,
  onDetail,
  onApprove
}: {
  capabilities: readonly Capability[];
  acting: boolean;
  t: ReturnType<typeof useRedesignI18n>["t"];
  onDetail: (capability: Capability) => void;
  onApprove: (capability: Capability) => void;
}) {
  return (
    <Card flush title={t("rd.cap.tableTitle")}>
      <Table
        caption={t("rd.cap.tableTitle")}
        empty={<EmptyState desc={t("rd.cap.emptyDesc")} title={t("rd.cap.empty")} />}
        renderCell={(row, column) => renderCapabilityCell(row, column, t, onDetail, onApprove, acting)}
        rowKey={(row) => row.id}
        rows={capabilities}
        tableId="adminCapabilities"
      />
    </Card>
  );
}

function renderCapabilityCell(
  row: Capability,
  column: CapabilityColumnKey,
  t: ReturnType<typeof useRedesignI18n>["t"],
  onDetail: (capability: Capability) => void,
  onApprove: (capability: Capability) => void,
  acting: boolean
) {
  switch (column) {
    case "capability":
      return (
        <div>
          <div className="cell-main">{row.displayName}</div>
          <div className="cell-sub mono">{row.key}</div>
        </div>
      );
    case "type":
      return <TagOutline>{t(`value.${row.action}`)}</TagOutline>;
    case "risk":
      return (
        <Chip tone={row.riskLevel === "critical" || row.riskLevel === "high" ? "danger" : row.riskLevel === "medium" ? "warning" : "neutral"}>
          {t(`rd.risk.${row.riskLevel}`)}
        </Chip>
      );
    case "dataScope":
      return (
        <span className="small">
          {row.dataDomains?.[0] ?? row.dataScopes?.find((scope) => scope.dataDomain)?.dataDomain ?? t("rd.cap.domainUnset")}
        </span>
      );
    case "status":
      return <Chip tone={capabilityStatusTone[row.discoveryStatus]}>{t(capabilityStatusKey[row.discoveryStatus])}</Chip>;
    case "actions":
      return (
        <div className="apr-actions">
          {row.discoveryStatus === "pending_review" ? (
            <Button disabled={acting} onClick={() => onApprove(row)} size="sm" variant="link">
              {t("rd.cap.approve")}
            </Button>
          ) : null}
          <Button onClick={() => onDetail(row)} size="sm" variant="link">
            {t("rd.cap.detail")}
          </Button>
        </div>
      );
  }
}
