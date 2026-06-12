import type { ReactNode } from "react";

import type { Translator } from "../consolePresenters";
import type { RoutePolicy } from "../types";
import { AccessPolicyWorkspace } from "./OperationalViews";

export function AiAdminView({ aiAdminPanel }: { aiAdminPanel: ReactNode }) {
  return (
    <section className="content-grid">
      {aiAdminPanel}
    </section>
  );
}

export function GettingStartedConsoleView({ gettingStartedPanel }: { gettingStartedPanel: ReactNode }) {
  return (
    <section className="content-grid">
      {gettingStartedPanel}
    </section>
  );
}

export function AskView({ askAccessPanel }: { askAccessPanel: ReactNode }) {
  return (
    <section className="content-grid">
      {askAccessPanel}
    </section>
  );
}

export function RegistryView({
  agentRegistryPanel,
  contractMatrixPanel,
  resourceLifecyclePanel
}: {
  agentRegistryPanel: ReactNode;
  contractMatrixPanel: ReactNode;
  resourceLifecyclePanel: ReactNode;
}) {
  return (
    <section className="content-grid">
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

export function AccessView({ accessProfilePanel }: { accessProfilePanel: ReactNode }) {
  return <section className="content-grid">{accessProfilePanel}</section>;
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
  managementAuditPanel,
  runtimeSignalsPanel
}: {
  evidenceRunsPanel: ReactNode;
  goLiveAcceptancePanel: ReactNode;
  managementAuditPanel: ReactNode;
  runtimeSignalsPanel: ReactNode;
}) {
  return (
    <section className="content-grid">
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
