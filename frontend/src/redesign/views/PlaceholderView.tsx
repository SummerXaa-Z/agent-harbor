import { ExternalLink } from "lucide-react";
import { useRedesignI18n } from "../hooks/useRedesignI18n";
import type { RedesignData } from "../hooks/useRedesignData";
import { legacyHashForView, viewLabelKey, type RedesignView } from "../router";
import { DataStatusBanner, DataStatusChip } from "../shell/DataStatusBanner";
import { Button } from "../ui/Button";
import { Card } from "../ui/Card";
import { KvList } from "../ui/KvList";
import { LoadingState } from "../ui/StateViews";

// Stands in for each page until its phase lands (P2–P4). It still shows real
// data status and hands the task to the legacy page that does the job today.
export function PlaceholderView({ data, onRetry, view }: { data: RedesignData; onRetry: () => void; view: RedesignView }) {
  const { t } = useRedesignI18n();
  const snapshot = data.data;

  return (
    <>
      <div className="page-head">
        <div className="page-head-main">
          <h1>{t(viewLabelKey(view))}</h1>
          <p>{t(`rd.page.${view}.desc`)}</p>
        </div>
        <div className="page-head-aside">
          <DataStatusChip status={data.status} />
        </div>
      </div>
      <DataStatusBanner data={data} onRetry={onRetry} />
      <div className="stack">
        <Card title={t("rd.placeholder.title")}>
          <div className="stack">
            <p className="muted">{t("rd.placeholder.pending")}</p>
            <div>
              <Button href={legacyHashForView[view]} icon={<ExternalLink aria-hidden="true" size={15} />} variant="ghost">
                {t("rd.placeholder.openLegacy")}
              </Button>
            </div>
          </div>
        </Card>
        {data.status === "error" ? null : (
          <Card title={t("rd.placeholder.summary")}>
            {snapshot ? (
              <KvList
                items={[
                  { key: "apiBase", label: t("rd.count.apiBase"), value: <span className="mono">{snapshot.apiBase}</span> },
                  { key: "agents", label: t("rd.count.agents"), value: snapshot.agents.length },
                  { key: "capabilities", label: t("rd.count.capabilities"), value: snapshot.capabilities.length },
                  { key: "tenants", label: t("rd.count.tenants"), value: snapshot.tenants.length },
                  { key: "routePolicies", label: t("rd.count.routePolicies"), value: snapshot.routePolicies.length },
                ]}
              />
            ) : (
              <LoadingState label={t("rd.data.loading")} />
            )}
          </Card>
        )}
      </div>
    </>
  );
}
