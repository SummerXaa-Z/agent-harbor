import { LogIn } from "lucide-react";
import type { FormEvent } from "react";
import type { ConsoleAuthMessage } from "../../hooks/useConsoleAuth";
import { tx } from "../../localizedMessages";
import { useRedesignI18n } from "../hooks/useRedesignI18n";
import { entryHash } from "../router";
import { Notice } from "../ui/Banner";
import { Button } from "../ui/Button";
import { Field } from "../ui/Field";
import { LogoIcon } from "../ui/icons";
import { LanguageToggle } from "./LanguageToggle";

interface LoginCardProps {
  loginKey: string;
  message: ConsoleAuthMessage | null;
  onLoginKeyChange: (value: string) => void;
  onSubmit: () => void;
  submitting: boolean;
}

const successMessageKey = "message.consoleLoginSucceeded";

export function LoginCard({ loginKey, message, onLoginKeyChange, onSubmit, submitting }: LoginCardProps) {
  const { t } = useRedesignI18n();
  const messageText = message ? (message.params ? tx(t, message.key, message.params) : t(message.key)) : "";
  const isError = Boolean(message && message.key !== successMessageKey);

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    onSubmit();
  }

  return (
    <div className="login">
      <div className="entry-tools">
        <LanguageToggle />
      </div>
      <section aria-labelledby="ah2-login-title" className="card login-card">
        <span className="entry-logo">
          <LogoIcon size={24} />
        </span>
        <h1 id="ah2-login-title">{t("auth.title")}</h1>
        <p>{t("auth.subtitle")}</p>
        <form className="signin-form" onSubmit={handleSubmit}>
          <Field
            error={isError ? messageText : undefined}
            hint={!isError && messageText ? messageText : undefined}
            htmlFor="ah2-admin-key"
            label={t("auth.adminKeyLabel")}
          >
            <input
              aria-invalid={isError}
              autoComplete="off"
              className="input"
              id="ah2-admin-key"
              onChange={(event) => onLoginKeyChange(event.target.value)}
              placeholder={t("auth.adminKeyPlaceholder")}
              spellCheck={false}
              type="password"
              value={loginKey}
            />
          </Field>
          <Button disabled={submitting} icon={<LogIn aria-hidden="true" size={15} />} type="submit">
            {submitting ? t("action.signingIn") : t("action.signIn")}
          </Button>
          <Notice>{t("auth.securityNote")}</Notice>
          <a className="text-link" href={entryHash}>
            {t("rd.login.backToEntry")}
          </a>
        </form>
      </section>
    </div>
  );
}
