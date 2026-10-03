import { RefreshCw } from "lucide-react";
import { useRedesignI18n } from "../hooks/useRedesignI18n";
import type { RedesignData } from "../hooks/useRedesignData";
import { apiErrorCopy, apiErrorPresentation } from "../model/apiErrorCategory";
import { Banner } from "../ui/Banner";
import { Button } from "../ui/Button";
import { Chip } from "../ui/Chip";

export function DataStatusChip({ status }: { status: RedesignData["status"] }) {
  const { t } = useRedesignI18n();
  if (status === "live") return <Chip tone="success">{t("rd.data.live")}</Chip>;
  if (status === "stale") return <Chip tone="warning">{t("rd.data.stale")}</Chip>;
  if (status === "sample") return <Chip tone="warning">{t("rd.data.sample")}</Chip>;
  return null;
}

// Sample rows, retained (possibly outdated) rows, and failed loads are always
// announced; live data needs no banner.
export function DataStatusBanner({ data, onRetry }: { data: RedesignData; onRetry: () => void }) {
  const { language, t } = useRedesignI18n();
  const retry = (
    <Button icon={<RefreshCw aria-hidden="true" size={14} />} onClick={onRetry} size="sm" variant="ghost">
      {t("rd.data.retry")}
    </Button>
  );

  if (data.status === "stale") {
    const copy = apiErrorCopy(t, "network");
    return <Banner actions={retry} desc={`${copy.next} ${t("rd.data.staleDesc")}`} title={copy.title} tone="warning" />;
  }
  if (data.status === "sample") {
    const copy = apiErrorCopy(t, "network");
    return <Banner actions={retry} desc={`${copy.next} ${t("rd.data.sampleDesc")}`} title={copy.title} tone="danger" />;
  }
  if (data.status === "error") {
    const presentation = apiErrorPresentation(t, language, data.error, "error.consoleDataUnavailable");
    return (
      <Banner
        actions={retry}
        desc={
          <>
            <div>{presentation.next}</div>
            <div className="mono">{presentation.detail}</div>
          </>
        }
        title={presentation.title}
        tone="danger"
      />
    );
  }
  return null;
}
