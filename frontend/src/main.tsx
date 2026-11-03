import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { loadTranslator } from "./i18n";
import { RootEntry } from "./RootEntry";
import { initialBrowserLanguage } from "./redesign/hooks/useRedesignI18n";
import { ErrorBoundary } from "./redesign/ui/ErrorBoundary";

// Top-level await: the active locale chunk resolves before the first render
// so no surface ever paints raw message keys, and only that one locale is
// fetched (the other stays an on-demand chunk until the user switches).
const initialLanguage = initialBrowserLanguage();
const initialTranslator = await loadTranslator(initialLanguage);

createRoot(document.getElementById("root") as HTMLElement).render(
  <StrictMode>
    <ErrorBoundary scope="app">
      <RootEntry initialI18n={{ language: initialLanguage, t: initialTranslator }} />
    </ErrorBoundary>
  </StrictMode>
);
