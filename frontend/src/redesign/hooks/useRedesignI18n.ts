import { createContext, useContext, useEffect, useMemo, useState } from "react";
import type { Translator } from "../../consolePresenters";
import { createTranslator, resolveInitialLanguage, type Language } from "../../i18n";

// Shared with the legacy console so switching between the two keeps the language.
const languageStorageKey = "agent-harbor-language";

export interface RedesignI18n {
  language: Language;
  setLanguage: (language: Language) => void;
  t: Translator;
}

export const RedesignI18nContext = createContext<RedesignI18n>({
  language: "en",
  setLanguage: () => undefined,
  t: createTranslator("en"),
});

export function useRedesignI18n(): RedesignI18n {
  return useContext(RedesignI18nContext);
}

function initialLanguage(): Language {
  const browserLanguages = Array.from(
    window.navigator.languages?.length ? window.navigator.languages : [window.navigator.language],
  ).filter(Boolean);
  try {
    return resolveInitialLanguage(window.localStorage.getItem(languageStorageKey), browserLanguages);
  } catch {
    return resolveInitialLanguage(undefined, browserLanguages);
  }
}

export function useRedesignLanguage(): RedesignI18n {
  const [language, setLanguage] = useState<Language>(initialLanguage);

  useEffect(() => {
    document.documentElement.lang = language;
    try {
      window.localStorage.setItem(languageStorageKey, language);
    } catch {
      // The UI still works when storage is unavailable.
    }
  }, [language]);

  const t = useMemo(() => createTranslator(language), [language]);
  return useMemo(() => ({ language, setLanguage, t }), [language, t]);
}
