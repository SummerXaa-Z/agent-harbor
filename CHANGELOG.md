# Changelog

All notable public changes to AgentHarbor will be documented in this file.

This project uses Keep a Changelog-style sections and semantic versioning for tagged releases.

## [Unreleased]

### Added

- Permission package templates now expose a stable version.
- Permission package application records are persisted in memory and PostgreSQL and returned from `POST /api/v1/permission-packages:apply`.
- `GET /api/v1/permission-packages/applications` lists permission package applications by tenant subtree, workspace, template, target, caller instance, and limit.
- Management MCP now exposes `list_permission_package_applications` for admin agents to review applied template versions, created assignment ids, capability ids, and data scopes.
- Permission package drafts now include a deterministic `policyGate` that allows direct apply for low-risk packages and requires approval for write, export, admin, high-risk, critical-risk, confidential, or restricted allowed capabilities.
- Web console AI Admin now shows the latest permission package application evidence after a successful apply.
- Web console AI Admin now shows policy-gate status and disables direct apply when a package requires approval.

## [0.1.0] - 2026-06-03

### Added

- Apache License 2.0 for open-source use.
- Public security policy for vulnerability reporting, supported versions, security scope, and disclosure handling.
- Product-oriented README covering AgentHarbor's tenant-first access control model, local setup, runtime configuration, API overview, policy semantics, and project docs.
- Local `.env.example` configuration template.
- Public roadmap for near-term and future product direction.
- Developer-preview core journey scenario with a dependency-free mock MCP server.
- Development-only `AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS` switch for local loopback/private upstream evaluation.
- English and Simplified Chinese labels for the web console's core tenant access and runtime evidence journey.
- Distinct web console workspaces for each primary navigation item so Cockpit, Registry, Routes, Policies, Capabilities, Access, Traces, and Evidence no longer collapse into the same view.
- Web console Core Journey Workbench that creates a fresh tenant tree, MCP target, scoped grant chain, allowed/denied runtime evidence, and tenant access profile through real APIs.
- `make mock-mcp` for keeping the dependency-free mock MCP server running during browser-based console evaluation.
- `make demo` for one-command local first-run evaluation of the API, mock MCP server, and web console.
- Core Journey Workbench preflight checks for API and Mock MCP readiness plus non-destructive demo session reset.

### Changed

- Reframed project documentation from internal development history to public open-source project documentation.
- Renamed the frontend package to `@agent-harbor/web-console`.
- Updated visible frontend title and brand copy to `AgentHarbor Control Plane`.
- Updated contribution, review, issue, PR, and release wording to use public project and scenario-script terminology.
- Renamed local smoke scripts to scenario naming.
- Renamed PostgreSQL migration files to stable schema-area filenames.
- Rewrote the frontend design reference as public console design guidance.
- Updated console metrics so live runtime evidence is shown from real allowed/denied traces instead of sample evidence runs.
- Added a visible web console language toggle that persists the operator's local preference.
- Expanded Simplified Chinese coverage for operator forms, tables, buttons, states, validation hints, and empty states in the web console.

### Fixed

- Prevented the Tenant Permission Console from blanking when access-profile collection fields are returned as `null`.
- Added accessible names and titles to icon-only primary navigation buttons.
- Made root-level frontend test/build targets install pinned frontend dependencies so `make check` works in a fresh clone.
- Made `make demo` install pinned frontend dependencies before starting the web console so a fresh clone can run the first browser evaluation without a separate install step.

### Removed

- Removed the outdated stacked PR runbook from public engineering docs.
- Removed internal planning and agent execution documents from public docs.
