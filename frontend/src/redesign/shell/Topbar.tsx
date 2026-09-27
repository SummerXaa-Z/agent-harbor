import { useRedesignI18n } from "../hooks/useRedesignI18n";
import { viewLabelKey, type RedesignView, type Surface } from "../router";
import { RefreshIcon } from "../ui/icons";
import { LanguageToggle } from "./LanguageToggle";
import { SurfaceSwitch } from "./SurfaceSwitch";

interface TopbarProps {
  onRefresh: () => void;
  refreshing: boolean;
  surface: Surface;
  view: RedesignView;
}

export function Topbar({ onRefresh, refreshing, surface, view }: TopbarProps) {
  const { t } = useRedesignI18n();
  return (
    <header className="tb">
      <div className="crumb">
        <span className="crumb-surface">{t(`rd.surface.${surface}`)}</span>
        <span aria-hidden="true" className="crumb-sep">
          /
        </span>
        <b>{t(viewLabelKey(view))}</b>
      </div>
      <div className="spacer" />
      <SurfaceSwitch placement="topbar" surface={surface} />
      <LanguageToggle />
      <button
        aria-busy={refreshing}
        aria-label={t("action.refresh")}
        className="icon-btn"
        disabled={refreshing}
        onClick={onRefresh}
        title={t("action.refresh")}
        type="button"
      >
        <RefreshIcon className={refreshing ? "spin" : undefined} />
      </button>
    </header>
  );
}
