import type { AdminIdentity, ConsoleSession } from "../../types.ts";

export const DEMO_REVIEWER_STORAGE_KEY = "agent-harbor-demo-reviewer";
export const DEFAULT_DEMO_REVIEWER = "security-reviewer";

export interface ReviewerResolution {
  reviewer: string;
  selectable: boolean;
}

// Demo mode (no login required) lets the operator pick who reviews, because
// the backend accepts any reviewer from `local-dev`. With login, the backend
// only accepts the signed-in actor, so the picker disappears.
export function resolveReviewer(session: ConsoleSession | null, stored?: string | null): ReviewerResolution {
  if (!session) return { reviewer: "", selectable: false };
  if (session.requiresLogin) {
    return { reviewer: session.actor?.trim() ?? "", selectable: false };
  }
  return { reviewer: stored?.trim() || DEFAULT_DEMO_REVIEWER, selectable: true };
}

export function reviewerOptions(admins: readonly AdminIdentity[]): string[] {
  const options = [DEFAULT_DEMO_REVIEWER];
  for (const admin of admins) {
    const actor = admin.actor.trim();
    if (admin.status === "active" && admin.role === "security_reviewer" && actor && !options.includes(actor)) {
      options.push(actor);
    }
  }
  return options;
}

export interface SessionIdentity {
  actor: string;
  canSignOut: boolean;
  demo: boolean;
  roleKey: string;
}

const knownRoles = new Set(["platform_admin", "security_reviewer", "tenant_admin"]);

// Demo mode runs as `local-dev` without a session, so the sidebar names it as
// the local dev admin and offers no sign-out; signed-in actors show as-is.
export function sessionIdentity(session: ConsoleSession | null): SessionIdentity {
  const demo = session?.requiresLogin === false;
  const role = session?.role?.trim() ?? "";
  return {
    actor: demo ? "" : session?.actor?.trim() ?? "",
    canSignOut: Boolean(session?.requiresLogin && session.authenticated),
    demo,
    roleKey: knownRoles.has(role) ? `auth.role.${role}` : "",
  };
}

// Segregation of duties: nobody approves their own request. The backend
// answers 403 as well; the UI disables the actions before that happens.
export function isSelfReview(reviewer?: string | null, requestedBy?: string | null): boolean {
  const normalizedReviewer = reviewer?.trim() ?? "";
  const normalizedRequester = requestedBy?.trim() ?? "";
  return normalizedReviewer !== "" && normalizedReviewer === normalizedRequester;
}
