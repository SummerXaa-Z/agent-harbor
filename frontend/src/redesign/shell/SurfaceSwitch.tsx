import { useRedesignI18n } from "../hooks/useRedesignI18n";
import { otherSurface, surfaceHomeHash, type Surface } from "../router";
import { SurfaceSwitchIcon } from "../ui/icons";

// Switching surfaces only switches views; it never grants a role (D1).
export function SurfaceSwitch({ placement, surface }: { placement: "sidebar" | "topbar"; surface: Surface }) {
  const { t } = useRedesignI18n();
  const target = otherSurface(surface);
  const label = t(target === "admin" ? "rd.switch.toAdmin" : "rd.switch.toUser");
  if (placement === "sidebar") {
    return (
      <a aria-label={label} className="sb-switch" href={surfaceHomeHash(target)} title={label}>
        <SurfaceSwitchIcon />
        <span className="sb-switch-label">{label}</span>
      </a>
    );
  }
  return (
    <a aria-label={label} className="app-switch" href={surfaceHomeHash(target)} title={label}>
      <SurfaceSwitchIcon size={14} />
      <span className="app-switch-label">{t(`rd.surface.${target}`)}</span>
    </a>
  );
}
