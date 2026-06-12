# Tenant Organization Workspace Design

## Goal

Make tenant management discoverable in the console and clarify that tenant permissions are managed through the existing permission-change workflow, not through hidden technical IDs or direct grant edits.

## Product Decision

AgentHarbor should expose tenants as a first-class administration context. The user should be able to answer two questions without guessing:

- Where do I manage tenants and organization scope?
- Where do I manage a tenant's permissions?

The console will add a `Tenants & Organization` workspace. It is read-only in this first slice and shows the tenant hierarchy, workspaces, registered agents, and permission summary for the selected tenant. Mutating permissions remains in `Permission Changes`, because approval, application, runtime validation, and go-live status checks must stay in one controlled journey.

## UX Shape

The page uses a simple B2B management layout:

- Left side: tenant hierarchy list with readable tenant names and level labels.
- Right side: selected tenant overview, workspace/agent counts, permission summary, and clear next actions.
- Primary action: start a permission change for the selected tenant.
- Secondary action: review effective access in the access profile page.

Technical IDs stay out of the main copy unless the source data has no business name.

## Architecture

Add a pure `tenantOrganization.ts` presenter that derives the hierarchy and selected-tenant summary from existing `ConsoleData` arrays. Add a focused `TenantOrganizationView` component that owns only local selection state and emits handoff context to `ConsoleController`.

`ConsoleController` only wires data and callbacks:

- `onStartPermissionChange(context)` sets the existing `permissionChange` handoff and opens `#ai-admin`.
- `onOpenAccessProfile(context)` updates the access profile filters and opens `#access`.

No backend API, schema, or data mutation changes are required.

## Visual Decision

Primary actions should be visibly filled enterprise-blue buttons. Secondary actions remain neutral outline buttons. This makes the main path legible without adding more theme colors. Semantic green/amber/red remain reserved for status.

## Acceptance

- Navigation includes `Tenants & Organization` / `租户与组织`.
- Getting Started directs tenant setup to the tenant workspace.
- Tenant page can select a tenant and start a permission change with tenant/workspace context preserved.
- Tenant page can open the access profile for the same tenant/workspace.
- Button styling makes `.primary-button` visibly stronger while preserving token usage and focus-visible behavior.
- Frontend tests, build, `make check`, and `make release-check` pass.
