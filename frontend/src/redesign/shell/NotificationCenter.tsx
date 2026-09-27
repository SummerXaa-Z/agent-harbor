import { useState } from "react";
import { tx } from "../../localizedMessages";
import { useRedesignI18n } from "../hooks/useRedesignI18n";
import type { NotificationItem } from "../model/notifications";
import type { Surface } from "../router";
import { BellIcon } from "../ui/icons";
import { Modal } from "../ui/Modal";
import { Button } from "../ui/Button";

interface NotificationCenterProps {
  items: readonly NotificationItem[];
  markAllRead: (surface: "admin" | "user") => void;
  markRead: (dedupeId: string) => void;
  surface: Surface;
  unread: number;
}

// Bell + panel. The bell counts the current surface's unread items (the
// other surface's count shows on the surface switch); the panel lists the
// current surface, marks single items or everything read, and follows the
// item's deep link.
export function NotificationCenter({ items, markAllRead, markRead, surface, unread }: NotificationCenterProps) {
  const { language, t } = useRedesignI18n();
  const [open, setOpen] = useState(false);
  const mine = items.filter((item) => item.surface === surface);

  return (
    <>
      <button
        aria-label={unread > 0 ? tx(t, "rd.nt.bell", { count: unread }) : t("rd.nt.title")}
        className="icon-btn"
        onClick={() => setOpen(true)}
        title={t("rd.nt.title")}
        type="button"
      >
        <BellIcon />
        {unread > 0 ? <span className="nt-badge">{unread > 99 ? "99+" : unread}</span> : null}
      </button>
      <Modal
        footer={
          <Button disabled={mine.length === 0} onClick={() => markAllRead(surface)} variant="ghost">
            {t("rd.nt.markAllRead")}
          </Button>
        }
        onClose={() => setOpen(false)}
        open={open}
        size="narrow"
        title={t("rd.nt.title")}
      >
        {mine.length === 0 ? (
          <p className="muted nt-empty">{t("rd.nt.empty")}</p>
        ) : (
          <ul className="nt-list">
            {mine.map((item) => (
              <li key={item.dedupeId}>
                <a
                  className="nt-item"
                  href={item.hash}
                  onClick={(event) => {
                    event.preventDefault();
                    markRead(item.dedupeId);
                    window.location.hash = item.hash;
                    setOpen(false);
                  }}
                >
                  <span aria-hidden="true" className={item.status === "pending" || item.kind === "env" ? "nt-dot nt-dot-live" : "nt-dot"} />
                  <span className="nt-main">
                    <span className="nt-title">{t(item.titleKey)}</span>
                    <span className="nt-sub">{tx(t, item.subKey, item.params)}</span>
                    <span className="nt-sub">
                      {t(`rd.nt.kind.${item.kind}`)}
                      {item.createdAt ? ` · ${formatNotificationTime(language, item.createdAt)}` : ""}
                    </span>
                  </span>
                </a>
              </li>
            ))}
          </ul>
        )}
      </Modal>
    </>
  );
}

function formatNotificationTime(language: string, value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString(language === "zh" ? "zh-CN" : "en-US", {
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    month: "numeric",
  });
}
