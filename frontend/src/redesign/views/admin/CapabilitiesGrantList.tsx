import { Button } from "../../ui/Button";
import { Card } from "../../ui/Card";
import { Chip } from "../../ui/Chip";
import { EmptyState } from "../../ui/StateViews";
import { tx } from "../../../localizedMessages";
import type { CapabilityGrantRow } from "../../model/capabilityCatalog";
import type { Capability } from "../../../types";
import type { useRedesignI18n } from "../../hooks/useRedesignI18n";

export function CapabilitiesGrantList({
  grants,
  capabilities,
  t,
  onManage
}: {
  grants: readonly CapabilityGrantRow[];
  capabilities: readonly Capability[];
  t: ReturnType<typeof useRedesignI18n>["t"];
  onManage: (entitlementId: string) => void;
}) {
  return (
    <Card title={t("rd.cap.grantsTitle")}>
      {grants.length === 0 ? (
        <EmptyState desc={t("rd.cap.grantsEmptyDesc")} title={t("rd.cap.grantsEmpty")} />
      ) : (
        <div className="cap-list">
          {grants.map((grant) => (
            <div className="cap-row" key={grant.entitlementId}>
              <div className="cap-main">
                <div className="cap-name">
                  {capabilities.find((capability) => capability.id === grant.capabilityId)?.displayName ?? grant.capabilityId}
                </div>
                <div className="cap-sub mono">
                  {grant.tenantId} · {tx(t, "rd.cap.grantWorkspaces", { count: grant.workspaceCount })} · {tx(t, "rd.cap.grantCallers", { count: grant.callerCount })}
                </div>
                {grant.subjectSelectors.length > 0 ? (
                  <div className="cap-sub mono">{grant.subjectSelectors.join(" · ")}</div>
                ) : null}
              </div>
              <div className="apr-button-row">
                <Chip tone={grant.effect === "deny" ? "danger" : "success"}>
                  {grant.effect === "deny" ? t("rd.decision.blocked") : t("rd.decision.allowed")}
                </Chip>
                <Button onClick={() => onManage(grant.entitlementId)} size="sm" variant="link">
                  {t("rd.cap.manageChain")}
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}
