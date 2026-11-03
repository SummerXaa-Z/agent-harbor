import type { ReactNode } from "react";
import type { Translator } from "../../src/consolePresenters";
import { RedesignI18nContext } from "../../src/redesign/hooks/useRedesignI18n";

// Keys pass through untranslated, so assertions match message keys instead
// of locale copy (tx still interpolates into whatever t returns).
export const stubT: Translator = (key) => key;

export function TestI18n({ children }: { children: ReactNode }) {
  return (
    <RedesignI18nContext.Provider value={{ language: "en", setLanguage: () => undefined, t: stubT }}>
      {children}
    </RedesignI18nContext.Provider>
  );
}
