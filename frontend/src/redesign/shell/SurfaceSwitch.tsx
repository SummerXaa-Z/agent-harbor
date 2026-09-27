import { useRedesignI18n } from "../hooks/useRedesignI18n";
import { otherSurface, surfaceHomeHash, type Surface } from "../router";
import { SurfaceSwitchIcon } from "../ui/icons";

// Switching surfaces only switches views; it never grants a role (D1). The
// badge carries the other surface's unread notification count (plan §8).
export function SurfaceSwitch({
  otherSurfaceUnread = 0,
  placement,
  surface
}: {
  otherSurfaceUnread?: number;
  placement: "sidebar" | "topbar";
  surface: Surface;
}) {
  const { t } = useRedesignI18n();
  const target = otherSurface(surface);
  const label = t(target === "admin" ? "rd.switch.toAdmin" : "rd.switch.toUser");
  const badge = otherSurfaceUnread > 0 ? (
    <span className="app-switch-badge">{otherSurfaceUnread > 99 ? "99+" : otherSurfaceUnread}</span>
  ) : null;
  if (placement === "sidebar") {
    return (
      <a aria-label={label} className="sb-switch" href={surfaceHomeHash(target)} title={label}>
        <SurfaceSwitchIcon />
        <span className="sb-switch-label">{label}</span>
        {badge}
      </a>
    );
  }
  return (
    <a aria-label={label} className="app-switch" href={surfaceHomeHash(target)} title={label}>
      <SurfaceSwitchIcon size={14} />
      <span className="app-switch-label">{t(`rd.surface.${target}`)}</span>
      {badge}
    </a>
  );
}
