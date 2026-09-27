import type { Language } from "../../i18n";
import { useRedesignI18n } from "../hooks/useRedesignI18n";

const languages: readonly { key: string; language: Language }[] = [
  { key: "rd.lang.zh", language: "zh-CN" },
  { key: "rd.lang.en", language: "en" },
];

export function LanguageToggle() {
  const { language, setLanguage, t } = useRedesignI18n();
  return (
    <div aria-label={t("control.language")} className="lang" role="group">
      {languages.map((option) => (
        <button
          aria-pressed={language === option.language}
          key={option.language}
          lang={option.language}
          onClick={() => setLanguage(option.language)}
          type="button"
        >
          {t(option.key)}
        </button>
      ))}
    </div>
  );
}
