# Frontend Design Reference

This document describes the intended product experience for the AgentHarbor web console. It is a public design reference for contributors, not a private implementation plan.

## Product Character

AgentHarbor is an enterprise control plane for agent and MCP governance. The console should feel like an operational tool for platform administrators, security owners, integration engineers, and auditors.

The first screen should be usable immediately. Avoid marketing-page patterns, oversized hero sections, decorative illustrations, and card-heavy layouts that slow down scanning. Prioritize dense but organized information, predictable navigation, and clear operational state.

## Information Architecture

- Use a stable application shell with primary navigation for Overview, Tenants, Agents, MCP Capabilities, Access Profiles, Route Policies, Audit, and Settings.
- Keep the top environment bar focused on tenant/workspace scope, API base, backend health, and operator context.
- Prefer tables and compact detail panels for operational objects. Include filtering, search, sorting, and stable row actions where the workflow needs them.
- Make object detail views lifecycle-oriented: configuration, permissions, linked callers/targets, recent activity, and audit evidence.
- Keep URLs stable enough for troubleshooting, review handoffs, and audit references.

## Governance Loop

Every major workflow should help operators answer four questions:

- Who is calling?
- Which tenant, workspace, and agent instance does the call belong to?
- Which MCP tool, route policy, or data scope made the decision possible?
- What evidence proves the allow or deny decision?

Allowed/denied, active/draft/revoked, healthy/degraded/unreachable, and inherited/overridden states should use consistent visual language across the console. Dangerous operations need clear confirmation, predictable side effects, and audit visibility.

## Interaction Principles

- Keep forms explicit. Tenant, workspace, caller instance, target, capability, and data scope controls should be visible when they affect authorization.
- Use familiar controls: tables for collections, tabs for related views, drawers or modals for focused edits, and icon buttons for compact actions.
- Show inherited tenant permissions separately from directly assigned permissions so administrators can reason about three-level tenant access.
- Treat empty states as actionable operational states: what is missing, which action creates it, and what permission is required.
- Do not hide policy consequences behind decorative language. Use precise labels like `allow`, `deny`, `inherited`, `direct`, `revoked`, and `restricted`.

## Accessibility and Maintainability

- Keep keyboard navigation usable for navigation, tables, filters, dialogs, drawers, and forms.
- Use semantic labels for controls, errors, badges, and status indicators.
- Do not rely on color alone for state. Pair color with text, icons, or badges.
- Desktop is the primary surface, but narrow screens should still support viewing, filtering, and the most important operations.
- Reuse the repository's existing component and token patterns before adding new visual primitives.
