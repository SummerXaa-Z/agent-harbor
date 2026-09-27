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

// The empty hash and every legacy hash keep the legacy console until the
// redesign replaces it (D1); `#/`, `#user/*` and `#admin/*` load the new UI.
export function RootEntry() {
  const hash = useSyncExternalStore(subscribeToHash, currentHash, () => "");
  return <Suspense fallback={null}>{isRedesignHash(hash) ? <RedesignApp hash={hash} /> : <LegacyEntry />}</Suspense>;
}
