import { useSyncExternalStore } from "react";
import RedesignApp from "./redesign/RedesignApp";

function subscribeToHash(onChange: () => void) {
  window.addEventListener("hashchange", onChange);
  return () => window.removeEventListener("hashchange", onChange);
}

function currentHash() {
  return window.location.hash;
}

// The legacy console is retired (D1): every hash — including retired legacy
// routes, which the router redirects — resolves inside the redesign.
export function RootEntry() {
  const hash = useSyncExternalStore(subscribeToHash, currentHash, () => "");
  return <RedesignApp hash={hash} />;
}
