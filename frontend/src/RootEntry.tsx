import { useSyncExternalStore } from "react";
import RedesignApp from "./redesign/RedesignApp";
import type { InitialI18n } from "./redesign/hooks/useRedesignI18n";

function subscribeToHash(onChange: () => void) {
  window.addEventListener("hashchange", onChange);
  return () => window.removeEventListener("hashchange", onChange);
}

function currentHash() {
  return window.location.hash;
}

// The legacy console is retired (D1): every hash — including retired legacy
// routes, which the router redirects — resolves inside the redesign. The
// initial translator is preloaded by main.tsx and flows in from there.
export function RootEntry({ initialI18n }: { initialI18n: InitialI18n }) {
  const hash = useSyncExternalStore(subscribeToHash, currentHash, () => "");
  return <RedesignApp hash={hash} initialI18n={initialI18n} />;
}
