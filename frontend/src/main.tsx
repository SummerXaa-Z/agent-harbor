import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { RootEntry } from "./RootEntry";

createRoot(document.getElementById("root") as HTMLElement).render(
  <StrictMode>
    <RootEntry />
  </StrictMode>
);
