import { LogOut } from "lucide-react";
import type { ConsoleSession } from "../../types";
import { useRedesignI18n } from "../hooks/useRedesignI18n";
import { sessionIdentity } from "../model/demoRole";
import {
  adminFooterNav,
  adminHash,
  adminNavSections,
  entryHash,
  userHash,
  userNavSections,
  type AdminView,
  type NavEntry,
  type RedesignView,
  type Surface,
  type UserView,
} from "../router";
import { LogoIcon, NavIcon } from "../ui/icons";
import { SurfaceSwitch } from "./SurfaceSwitch";

interface SidebarProps {
  activeView: RedesignView;
  onSignOut: () => void;
  otherSurfaceUnread: number;
  session: ConsoleSession | null;
  surface: Surface;
}

export function Sidebar({ activeView, onSignOut, otherSurfaceUnread, session, surface }: SidebarProps) {
  const { t } = useRedesignI18n();
  const sections = surface === "user" ? userNavSections : adminNavSections;
  const identity = sessionIdentity(session);
  const name = identity.demo ? t("rd.demo.localAdmin") : identity.actor || t("auth.unknownActor");
  const role = identity.roleKey ? t(identity.roleKey) : "";

  return (
    <aside className="sb">
      <a aria-label="AgentHarbor" className="sb-brand" href={entryHash}>
        <span className="sb-logo">
          <LogoIcon />
        </span>
        <div className="sb-brand-text">
          <div className="sb-name">AgentHarbor</div>
          <div className="sb-tag">{t(`rd.surface.${surface}`)}</div>
        </div>
      </a>
      <nav aria-label={t("rd.shell.primaryNav")} className="sb-nav">
        {sections.map((section, index) => (
          <div key={section.labelKey ?? index}>
            {section.labelKey ? <div className="sb-group-label">{t(section.labelKey)}</div> : null}
            {section.items.map((item) => (
              <NavLink activeView={activeView} item={item} key={item.view} surface={surface} />
            ))}
          </div>
        ))}
      </nav>
      <div className="sb-bottom">
        {surface === "admin" ? <NavLink activeView={activeView} item={adminFooterNav} surface={surface} /> : null}
        <SurfaceSwitch placement="sidebar" surface={surface} />
        <div aria-label={t("rd.shell.currentUser")} className="sb-user" role="group">
          <span aria-hidden="true" className="avatar">
            {Array.from(name)[0]?.toUpperCase()}
          </span>
          <div className="u-info">
            <div className="u-name" title={name}>
              {name}
            </div>
            {role ? <div className="u-role">{role}</div> : null}
          </div>
          {identity.canSignOut ? (
            <button
              aria-label={t("action.signOut")}
              className="icon-btn sm"
              onClick={onSignOut}
              title={t("action.signOut")}
              type="button"
            >
              <LogOut aria-hidden="true" size={15} />
            </button>
          ) : null}
        </div>
      </div>
    </aside>
  );
}

function NavLink({
  activeView,
  item,
  surface,
}: {
  activeView: RedesignView;
  item: NavEntry;
  surface: Surface;
}) {
  const { t } = useRedesignI18n();
  const label = t(item.labelKey);
  const active = item.view === activeView;
  const href = surface === "user" ? userHash(item.view as UserView) : adminHash(item.view as AdminView);
  return (
    <a
      aria-current={active ? "page" : undefined}
      aria-label={label}
      className={active ? "nav-link active" : "nav-link"}
      href={href}
      title={label}
    >
      <NavIcon view={item.view} />
      <span className="nav-label">{label}</span>
    </a>
  );
}
