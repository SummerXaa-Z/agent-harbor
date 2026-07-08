import type { ReactNode } from "react";
import { ChevronRight } from "lucide-react";

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
  resourceLifecyclePanel,
  t
}: {
  agentRegistryPanel: ReactNode;
  contractMatrixPanel: ReactNode;
  journeyCheckpoint?: ReactNode;
  resourceLifecyclePanel: ReactNode;
  t: Translator;
}) {
  return (
    <section className="content-grid">
      {journeyCheckpoint}
      {resourceLifecyclePanel}
      <details className="resource-advanced-details">
        <summary>
          <span>
            <strong>{t("resource.advanced.title")}</strong>
            <small>{t("resource.advanced.detail")}</small>
          </span>
          <span className="resource-advanced-action">
            {t("resource.advanced.action")}
            <ChevronRight size={15} aria-hidden="true" />
          </span>
        </summary>
        <div className="resource-advanced-grid">
          {agentRegistryPanel}
          {contractMatrixPanel}
        </div>
      </details>
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

export function AdminAccessView({ adminAccessPanel }: { adminAccessPanel: ReactNode }) {
  return (
    <section className="content-grid">
      {adminAccessPanel}
    </section>
  );
}

export function TracesView({
  managementAuditEventCount,
  managementAuditPanel,
  t,
  tracePanel
}: {
  managementAuditEventCount: number;
  managementAuditPanel: ReactNode;
  t: Translator;
  tracePanel: ReactNode;
}) {
  return (
    <section className="content-grid">
      {tracePanel}
      {managementAuditEventCount > 0 ? (
        managementAuditPanel
      ) : (
        <details className="resource-advanced-details trace-audit-disclosure">
          <summary>
            <span>
              <strong>{t("panel.managementAudit")}</strong>
              <small>{t("empty.managementAudit.detail")}</small>
            </span>
            <span className="resource-advanced-action">
              {t("action.showAudit")}
              <ChevronRight size={15} aria-hidden="true" />
            </span>
          </summary>
          <div className="trace-audit-disclosure-body">
            {managementAuditPanel}
          </div>
        </details>
      )}
    </section>
  );
}

export function GoLiveStatusView({
  acceptanceHistoryPanel,
  goLiveAcceptancePanel,
  journeyCheckpoint,
  managementAuditPanel,
  runtimeSignalsPanel
}: {
  acceptanceHistoryPanel: ReactNode;
  goLiveAcceptancePanel: ReactNode;
  journeyCheckpoint?: ReactNode;
  managementAuditPanel: ReactNode;
  runtimeSignalsPanel: ReactNode;
}) {
  return (
    <section className="content-grid">
      {journeyCheckpoint}
      {goLiveAcceptancePanel}
      {acceptanceHistoryPanel}
      {managementAuditPanel}
      {runtimeSignalsPanel}
    </section>
  );
}

export function CockpitView({
  coreJourneyPanel,
  acceptanceHistoryPanel,
  runtimeSignalsPanel,
  tracePanel
}: {
  coreJourneyPanel: ReactNode;
  acceptanceHistoryPanel: ReactNode;
  runtimeSignalsPanel: ReactNode;
  tracePanel: ReactNode;
}) {
  return (
    <section className="content-grid">
      {coreJourneyPanel}
      {runtimeSignalsPanel}
      {tracePanel}
      {acceptanceHistoryPanel}
    </section>
  );
}
