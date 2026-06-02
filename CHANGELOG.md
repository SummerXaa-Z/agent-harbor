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

### Changed

- Reframed project documentation from internal development history to public open-source project documentation.
- Renamed the frontend package to `@agent-harbor/web-console`.
- Updated visible frontend title and brand copy to `AgentHarbor Control Plane`.
- Updated contribution, review, issue, PR, and release wording to use public project and scenario-script terminology.
- Renamed local smoke scripts to scenario naming.
- Renamed PostgreSQL migration files to stable schema-area filenames.
- Rewrote the frontend design reference as public console design guidance.

### Removed

- Removed the outdated stacked PR runbook from public engineering docs.
- Removed internal planning and agent execution documents from public docs.
