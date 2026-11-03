import { createContext, useContext, useEffect, useMemo, useRef, useState } from "react";
import type { Translator } from "../../consolePresenters";
import { loadTranslator, resolveInitialLanguage, type Language } from "../../i18n";

// Shared with the legacy console so switching between the two keeps the language.
const languageStorageKey = "agent-harbor-language";

export interface RedesignI18n {
  language: Language;
  setLanguage: (language: Language) => void;
  t: Translator;
}

// Preloaded by main.tsx before the first render (see loadTranslator).
export type InitialI18n = Pick<RedesignI18n, "language" | "t">;

// Outside a provider (tests, tooling) keys pass through untranslated.
export const RedesignI18nContext = createContext<RedesignI18n>({
  language: "en",
  setLanguage: () => undefined,
  t: (key) => key,
});

export function useRedesignI18n(): RedesignI18n {
  return useContext(RedesignI18nContext);
}

export function initialBrowserLanguage(): Language {
  const browserLanguages = Array.from(
    window.navigator.languages?.length ? window.navigator.languages : [window.navigator.language],
  ).filter(Boolean);
  try {
    return resolveInitialLanguage(window.localStorage.getItem(languageStorageKey), browserLanguages);
  } catch {
    return resolveInitialLanguage(undefined, browserLanguages);
  }
}

// The initial translator arrives preloaded from main.tsx so the first paint
// is never raw keys; switching languages keeps the old translator until the
// new locale chunk resolves (no untranslated flash mid-session).
export function useRedesignLanguage(initial: { language: Language; t: Translator }): RedesignI18n {
  const [language, setLanguage] = useState<Language>(initial.language);
  // Translators are functions, so both the initializer and the setter must
  // wrap them: a bare function argument would be treated as React's lazy
  // initializer / state updater and get called instead of stored.
  const [t, setT] = useState<Translator>(() => initial.t);
  const loadedFor = useRef(initial.language);

  useEffect(() => {
    document.documentElement.lang = language;
    try {
      window.localStorage.setItem(languageStorageKey, language);
    } catch {
      // The UI still works when storage is unavailable.
    }
  }, [language]);

  useEffect(() => {
    if (loadedFor.current === language) return;
    let cancelled = false;
    loadTranslator(language).then(
      (next) => {
        if (cancelled) return;
        loadedFor.current = language;
        setT(() => next);
      },
      (error) => {
        console.error("failed to load translations", error);
      },
    );
    return () => {
      cancelled = true;
    };
  }, [language]);

  return useMemo(() => ({ language, setLanguage, t }), [language, t]);
}
