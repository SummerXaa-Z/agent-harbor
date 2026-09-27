import { Compass } from "lucide-react";
import { tx } from "../../localizedMessages";
import { useRedesignI18n } from "../hooks/useRedesignI18n";
import { entryHash, userHash } from "../router";
import { Button } from "../ui/Button";

// Rendered for slash-shaped hashes that match no redesign view (round 4,
// #27). The attempted hash stays in the address bar so it can be inspected.
export function NotFoundView({ attempted }: { attempted: string }) {
  const { t } = useRedesignI18n();
  return (
    <div className="login">
      <section aria-labelledby="ah2-notfound-title" className="card login-card">
        <Compass aria-hidden="true" size={28} />
        <h1 id="ah2-notfound-title">{t("rd.notfound.title")}</h1>
        <p className="muted">{tx(t, "rd.notfound.desc", { hash: attempted })}</p>
        <div className="signin-form">
          <Button href={userHash("home")}>{t("rd.notfound.backHome")}</Button>
          <a className="text-link" href={entryHash}>{t("rd.notfound.entry")}</a>
        </div>
      </section>
    </div>
  );
}
