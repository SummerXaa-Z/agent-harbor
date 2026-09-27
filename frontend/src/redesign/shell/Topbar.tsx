import { useRedesignI18n } from "../hooks/useRedesignI18n";
import type { NotificationsState } from "../hooks/useNotifications";
import { viewLabelKey, type RedesignView, type Surface } from "../router";
import { RefreshIcon, SearchIcon } from "../ui/icons";
import { LanguageToggle } from "./LanguageToggle";
import { NotificationCenter } from "./NotificationCenter";
import { SurfaceSwitch } from "./SurfaceSwitch";

interface TopbarProps {
  notifications: NotificationsState;
  onOpenPalette: () => void;
  onRefresh: () => void;
  otherSurfaceUnread: number;
  refreshing: boolean;
  surface: Surface;
  view: RedesignView;
}

export function Topbar({
  notifications,
  onOpenPalette,
  onRefresh,
  otherSurfaceUnread,
  refreshing,
  surface,
  view
}: TopbarProps) {
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
      <SurfaceSwitch otherSurfaceUnread={otherSurfaceUnread} placement="topbar" surface={surface} />
      <LanguageToggle />
      <NotificationCenter
        items={notifications.items}
        markAllRead={notifications.markAllRead}
        markRead={notifications.markRead}
        surface={surface}
        unread={notifications.unread[surface]}
      />
      <button
        aria-label={t("rd.palette.title")}
        className="icon-btn tb-kbtn"
        onClick={onOpenPalette}
        title={t("rd.palette.title")}
        type="button"
      >
        <SearchIcon />
        <span className="tb-kbtn-text">{t("rd.palette.open")}</span>
        <kbd className="tb-kbd">⌘K</kbd>
      </button>
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
