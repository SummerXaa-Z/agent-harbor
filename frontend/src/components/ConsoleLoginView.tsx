import { LockKeyhole, LogIn, Network } from "lucide-react";
import type { FormEvent } from "react";
import type { Translator } from "../consolePresenters";
import type { Language } from "../i18n";

interface ConsoleLoginViewProps {
  adminKey: string;
  language: Language;
  loading: boolean;
  message: string;
  onAdminKeyChange: (value: string) => void;
  onLanguageChange: (language: Language) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  t: Translator;
}

export function ConsoleLoginView({
  adminKey,
  language,
  loading,
  message,
  onAdminKeyChange,
  onLanguageChange,
  onSubmit,
  t
}: ConsoleLoginViewProps) {
  return (
    <main className="login-shell">
      <div className="login-language">
        <div className="language-toggle" aria-label={t("control.language")} role="group">
          <button
            className={language === "zh-CN" ? "selected" : ""}
            onClick={() => onLanguageChange("zh-CN")}
            type="button"
          >
            中文
          </button>
          <button
            className={language === "en" ? "selected" : ""}
            onClick={() => onLanguageChange("en")}
            type="button"
          >
            EN
          </button>
        </div>
      </div>

      <section className="login-panel" aria-labelledby="console-login-title">
        <div className="login-brand">
          <span className="login-brand-mark">
            <Network size={19} />
          </span>
          <div>
            <strong>AgentHarbor</strong>
            <span>{t("app.controlPlane")}</span>
          </div>
        </div>

        <div className="login-copy">
          <span className="section-kicker">{t("auth.kicker")}</span>
          <h1 id="console-login-title">{t("auth.title")}</h1>
          <p>{t("auth.subtitle")}</p>
        </div>

        <form className="login-form" onSubmit={onSubmit}>
          <label className="login-field">
            <span>{t("auth.adminKeyLabel")}</span>
            <input
              autoComplete="current-password"
              autoFocus
              onChange={(event) => onAdminKeyChange(event.target.value)}
              placeholder={t("auth.adminKeyPlaceholder")}
              type="password"
              value={adminKey}
            />
          </label>

          {message ? (
            <div className="login-message" role="status">
              <LockKeyhole size={15} />
              <span>{message}</span>
            </div>
          ) : null}

          <button className="primary-button login-submit" disabled={loading} type="submit">
            <LogIn size={16} />
            {loading ? t("action.signingIn") : t("action.signIn")}
          </button>
        </form>

        <div className="login-footnote">
          <LockKeyhole size={15} />
          <span>{t("auth.securityNote")}</span>
        </div>
      </section>
    </main>
  );
}
