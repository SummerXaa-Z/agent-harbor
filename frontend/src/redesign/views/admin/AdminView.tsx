import { lazy } from "react";
import type { AdminView as AdminViewName } from "../../router";
import type { AdminViewProps } from "./adminViewProps";

// Each admin page loads as its own chunk; the shell Suspense in RedesignApp
// covers the brief gap while a page chunk streams in.
const AdminAccessView = lazy(() => import("./AdminAccessView").then((m) => ({ default: m.AdminAccessView })));
const ApprovalsView = lazy(() => import("./ApprovalsView").then((m) => ({ default: m.ApprovalsView })));
const CapabilitiesView = lazy(() => import("./CapabilitiesView").then((m) => ({ default: m.CapabilitiesView })));
const CockpitView = lazy(() => import("./CockpitView").then((m) => ({ default: m.CockpitView })));
const PoliciesView = lazy(() => import("./PoliciesView").then((m) => ({ default: m.PoliciesView })));
const RegistryView = lazy(() => import("./RegistryView").then((m) => ({ default: m.RegistryView })));
const RoutesView = lazy(() => import("./RoutesView").then((m) => ({ default: m.RoutesView })));
const TenantsView = lazy(() => import("./TenantsView").then((m) => ({ default: m.TenantsView })));
const TracesView = lazy(() => import("./TracesView").then((m) => ({ default: m.TracesView })));

// Cockpit, approvals and traces landed in P3; the six governance pages
// (tenants, registry, capabilities, policies, routes, admin boundaries)
// landed in P4. The switch is exhaustive over AdminViewName.
export function AdminView({ view, ...props }: AdminViewProps & { view: AdminViewName }) {
  switch (view) {
    case "cockpit":
      return <CockpitView {...props} />;
    case "approvals":
      return <ApprovalsView {...props} />;
    case "traces":
      return <TracesView {...props} />;
    case "tenants":
      return <TenantsView {...props} />;
    case "registry":
      return <RegistryView {...props} />;
    case "capabilities":
      return <CapabilitiesView {...props} />;
    case "policies":
      return <PoliciesView {...props} />;
    case "routes":
      return <RoutesView {...props} />;
    case "admin":
      return <AdminAccessView {...props} />;
  }
}
