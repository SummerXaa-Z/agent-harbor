# Changelog

All notable public changes to AgentHarbor will be documented in this file.

This project uses Keep a Changelog-style sections and will adopt semantic versioning once tagged releases begin.

## [Unreleased]

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

### Removed

- Removed the outdated stacked PR runbook from public engineering docs.
- Removed internal planning and agent execution documents from public docs.
