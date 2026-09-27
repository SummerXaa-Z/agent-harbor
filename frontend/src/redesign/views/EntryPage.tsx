import type { ReactNode } from "react";
import { tx } from "../../localizedMessages";
import type { ConsoleSession } from "../../types";
import { useRedesignI18n } from "../hooks/useRedesignI18n";
import { LanguageToggle } from "../shell/LanguageToggle";
import { surfaceHomeHash, type Surface } from "../router";
import { AdminSurfaceIcon, GoArrowIcon, ListCheckIcon, LogoIcon, UserSurfaceIcon } from "../ui/icons";

const entryItems = ["item1", "item2", "item3", "item4"] as const;

export function EntryPage({ apiBase, session }: { apiBase: string; session: ConsoleSession | null }) {
  const { t } = useRedesignI18n();
  const modeKey = session?.requiresLogin === false ? "rd.entry.footDemo" : session ? "rd.entry.footLogin" : "";

  return (
    <div className="entry">
      <div className="entry-tools">
        <LanguageToggle />
      </div>
      <div className="entry-brand">
        <span className="entry-logo">
          <LogoIcon size={24} />
        </span>
        <h1>AgentHarbor</h1>
      </div>
      <p className="entry-sub">{t("rd.entry.subtitle")}</p>
      <div className="entry-cards">
        <EntryCard icon={<UserSurfaceIcon />} surface="user" />
        <EntryCard icon={<AdminSurfaceIcon />} surface="admin" />
      </div>
      <p className="entry-foot">
        {tx(t, "rd.entry.foot", { apiBase })}
        {modeKey ? (
          <>
            <br />
            {t(modeKey)}
          </>
        ) : null}
      </p>
      <a className="entry-legacy" href="#getting-started">
        {t("rd.entry.legacy")}
      </a>
    </div>
  );
}

function EntryCard({ icon, surface }: { icon: ReactNode; surface: Surface }) {
  const { t } = useRedesignI18n();
  return (
    <section className="entry-card">
      <span className="entry-ico">{icon}</span>
      <h2>{t(`rd.surface.${surface}`)}</h2>
      <p className="entry-desc">{t(`rd.entry.${surface}.desc`)}</p>
      <ul>
        {entryItems.map((item) => (
          <li key={item}>
            <ListCheckIcon />
            {t(`rd.entry.${surface}.${item}`)}
          </li>
        ))}
      </ul>
      <a className={surface === "user" ? "entry-go" : "entry-go entry-go-ghost"} href={surfaceHomeHash(surface)}>
        {t(`rd.entry.${surface}.go`)}
        <GoArrowIcon />
      </a>
    </section>
  );
}
