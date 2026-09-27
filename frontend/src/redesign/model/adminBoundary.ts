import type { AdminIdentity, ConsoleSession } from "../../types";

// Admin & boundaries page view-model. Separation-of-duties and production
// boundaries are enforced by the backend and policy gates — the console only
// renders them read-only (plan difference #12).

export type AdminRoleKey = "platform_admin" | "tenant_admin" | "security_reviewer";

export interface AdminBoundaryRow {
  displayName: string;
  identity: AdminIdentity;
  scope: { tenantId: string; workspaceId: string };
}

export function adminBoundaryRows(identities: readonly AdminIdentity[]): AdminBoundaryRow[] {
  return identities
    .map((identity): AdminBoundaryRow => ({
      displayName: identity.displayName || identity.actor,
      identity,
      scope: { tenantId: identity.tenantId ?? "", workspaceId: identity.workspaceId ?? "" }
    }))
    .sort((a, b) => a.displayName.localeCompare(b.displayName));
}

export interface AdminBoundarySummary {
  active: number;
  bootstrap: number;
  disabled: number;
  managed: number;
}

export function adminBoundarySummary(identities: readonly AdminIdentity[]): AdminBoundarySummary {
  return {
    active: identities.filter((identity) => identity.status === "active").length,
    bootstrap: identities.filter((identity) => identity.source === "bootstrap").length,
    disabled: identities.filter((identity) => identity.status === "disabled").length,
    managed: identities.filter((identity) => identity.source === "managed").length
  };
}

// Demo sessions (backend started with auth off) get one extra row naming the
// local development admin driving the console.
export function demoAdminSessionVisible(session: ConsoleSession | null): boolean {
  // Demo mode = the session never needed a login (local dev API). The
  // local-dev actor may still report authenticated=true, which is exactly the
  // auto-granted session the row is meant to surface.
  return session !== null && session.requiresLogin === false;
}

// Only managed identities can be rotated or disabled; bootstrap identities
// are the break-glass account the deployment owns.
export function adminIdentityMutable(identity: AdminIdentity): boolean {
  return identity.source === "managed";
}

export function createAdminIdentityRequestReady(input: {
  actor: string;
  role: AdminRoleKey;
  tenantId: string;
}): boolean {
  if (!input.actor.trim()) return false;
  return input.role === "platform_admin" || Boolean(input.tenantId.trim());
}
