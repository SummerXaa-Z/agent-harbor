import { PlaceholderView } from "../PlaceholderView";
import type { AdminView as AdminViewName } from "../../router";
import type { AdminViewProps } from "./adminViewProps";
import { ApprovalsView } from "./ApprovalsView";
import { CockpitView } from "./CockpitView";
import { TracesView } from "./TracesView";

// Cockpit, approvals and traces are redesigned (P3); the governance pages stay
// on placeholders until P4 and keep their link into the legacy console.
export function AdminView({ view, ...props }: AdminViewProps & { view: AdminViewName }) {
  switch (view) {
    case "cockpit":
      return <CockpitView {...props} />;
    case "approvals":
      return <ApprovalsView {...props} />;
    case "traces":
      return <TracesView {...props} />;
    default:
      return <PlaceholderView data={props.data} onRetry={props.onRetry} view={view} />;
  }
}
