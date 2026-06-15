import type { ReactNode } from "react";

import type { Translator } from "../consolePresenters";
import type { RoutePolicy } from "../types";
import { AccessPolicyWorkspace } from "./OperationalViews";

export function AiAdminView({
  aiAdminPanel,
  journeyCheckpoint
}: {
  aiAdminPanel: ReactNode;
  journeyCheckpoint?: ReactNode;
}) {
  return (
    <section className="content-grid">
      {journeyCheckpoint}
      {aiAdminPanel}
    </section>
  );
}

export function GettingStartedConsoleView({
  gettingStartedPanel,
  journeyCheckpoint
}: {
  gettingStartedPanel: ReactNode;
  journeyCheckpoint?: ReactNode;
}) {
  return (
    <section className="content-grid">
      {journeyCheckpoint}
      {gettingStartedPanel}
    </section>
  );
}

export function AskView({
  askAccessPanel,
  journeyCheckpoint
}: {
  askAccessPanel: ReactNode;
  journeyCheckpoint?: ReactNode;
}) {
  return (
    <section className="content-grid">
      {journeyCheckpoint}
      {askAccessPanel}
    </section>
  );
}

export function RegistryView({
  agentRegistryPanel,
  contractMatrixPanel,
  journeyCheckpoint,
  resourceLifecyclePanel
}: {
  agentRegistryPanel: ReactNode;
  contractMatrixPanel: ReactNode;
  journeyCheckpoint?: ReactNode;
  resourceLifecyclePanel: ReactNode;
}) {
  return (
    <section className="content-grid">
      {journeyCheckpoint}
      {resourceLifecyclePanel}
      {agentRegistryPanel}
      {contractMatrixPanel}
    </section>
  );
}

export function RoutesView({
  routeGovernancePanel,
  tracePanel
}: {
  routeGovernancePanel: ReactNode;
  tracePanel: ReactNode;
}) {
  return (
    <section className="content-grid">
      {routeGovernancePanel}
      {tracePanel}
    </section>
  );
}

export function PoliciesView({
  capabilityGovernancePanel,
  managementAuditPanel,
  policies,
  routeGovernancePanel,
  t
}: {
  capabilityGovernancePanel: ReactNode;
  managementAuditPanel: ReactNode;
  policies: RoutePolicy[];
  routeGovernancePanel: ReactNode;
  t: Translator;
}) {
  return (
    <section className="content-grid">
      <AccessPolicyWorkspace
        managementAuditPanel={managementAuditPanel}
        policies={policies}
        routeGovernancePanel={routeGovernancePanel}
        t={t}
      />
      {capabilityGovernancePanel}
    </section>
  );
}

export function CapabilitiesView({ capabilityGovernancePanel }: { capabilityGovernancePanel: ReactNode }) {
  return (
    <section className="content-grid">
      {capabilityGovernancePanel}
    </section>
  );
}

export function AccessView({
  accessProfilePanel,
  journeyCheckpoint
}: {
  accessProfilePanel: ReactNode;
  journeyCheckpoint?: ReactNode;
}) {
  return (
    <section className="content-grid">
      {journeyCheckpoint}
      {accessProfilePanel}
    </section>
  );
}

export function TenantsView({ tenantOrganizationPanel }: { tenantOrganizationPanel: ReactNode }) {
  return <>{tenantOrganizationPanel}</>;
}

export function TracesView({
  managementAuditPanel,
  tracePanel
}: {
  managementAuditPanel: ReactNode;
  tracePanel: ReactNode;
}) {
  return (
    <section className="content-grid">
      {tracePanel}
      {managementAuditPanel}
    </section>
  );
}

export function EvidenceView({
  evidenceRunsPanel,
  goLiveAcceptancePanel,
  journeyCheckpoint,
  managementAuditPanel,
  runtimeSignalsPanel
}: {
  evidenceRunsPanel: ReactNode;
  goLiveAcceptancePanel: ReactNode;
  journeyCheckpoint?: ReactNode;
  managementAuditPanel: ReactNode;
  runtimeSignalsPanel: ReactNode;
}) {
  return (
    <section className="content-grid">
      {journeyCheckpoint}
      {goLiveAcceptancePanel}
      {evidenceRunsPanel}
      {managementAuditPanel}
      {runtimeSignalsPanel}
    </section>
  );
}

export function CockpitView({
  agentRegistryPanel,
  coreJourneyPanel,
  evidenceRunsPanel,
  runtimeSignalsPanel,
  tracePanel
}: {
  agentRegistryPanel: ReactNode;
  coreJourneyPanel: ReactNode;
  evidenceRunsPanel: ReactNode;
  runtimeSignalsPanel: ReactNode;
  tracePanel: ReactNode;
}) {
  return (
    <section className="content-grid">
      {coreJourneyPanel}
      {runtimeSignalsPanel}
      {tracePanel}
      {evidenceRunsPanel}
      {agentRegistryPanel}
    </section>
  );
}
