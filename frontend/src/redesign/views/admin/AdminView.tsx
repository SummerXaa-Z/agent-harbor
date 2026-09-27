import { PlaceholderView } from "../PlaceholderView";
import type { AdminView as AdminViewName } from "../../router";
import type { AdminViewProps } from "./adminViewProps";
import { AdminAccessView } from "./AdminAccessView";
import { ApprovalsView } from "./ApprovalsView";
import { CapabilitiesView } from "./CapabilitiesView";
import { CockpitView } from "./CockpitView";
import { PoliciesView } from "./PoliciesView";
import { RegistryView } from "./RegistryView";
import { RoutesView } from "./RoutesView";
import { TenantsView } from "./TenantsView";
import { TracesView } from "./TracesView";

// Cockpit, approvals and traces landed in P3; the six governance pages
// (tenants, registry, capabilities, policies, routes, admin boundaries)
// landed in P4. Nothing here falls back to the legacy console anymore.
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
    default:
      return <PlaceholderView data={props.data} onRetry={props.onRetry} view={view} />;
  }
}
