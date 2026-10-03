import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { RootEntry } from "./RootEntry";
import { ErrorBoundary } from "./redesign/ui/ErrorBoundary";

createRoot(document.getElementById("root") as HTMLElement).render(
  <StrictMode>
    <ErrorBoundary scope="app">
      <RootEntry />
    </ErrorBoundary>
  </StrictMode>
);
