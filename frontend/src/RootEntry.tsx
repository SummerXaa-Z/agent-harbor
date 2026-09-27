import { lazy, Suspense, useSyncExternalStore } from "react";
import { isRedesignHash } from "./redesign/router.ts";

const RedesignApp = lazy(() => import("./redesign/RedesignApp"));
const LegacyEntry = lazy(() => import("./LegacyEntry"));

function subscribeToHash(onChange: () => void) {
  window.addEventListener("hashchange", onChange);
  return () => window.removeEventListener("hashchange", onChange);
}

function currentHash() {
  return window.location.hash;
}

// Since P5 the empty hash loads the redesign entry page (D1); explicit
// legacy hashes (#ask, #getting-started, …) keep the legacy console.
export function RootEntry() {
  const hash = useSyncExternalStore(subscribeToHash, currentHash, () => "");
  return <Suspense fallback={null}>{isRedesignHash(hash) ? <RedesignApp hash={hash} /> : <LegacyEntry />}</Suspense>;
}
