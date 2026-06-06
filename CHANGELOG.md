# Changelog

All notable public changes to AgentHarbor will be documented in this file.

This project uses Keep a Changelog-style sections and semantic versioning for tagged releases.

## [Unreleased]

### Added

- Permission package templates now expose a stable version.
- Permission package application records are persisted in memory and PostgreSQL and returned from `POST /api/v1/permission-packages:apply`.
- `GET /api/v1/permission-packages/applications` lists permission package applications by tenant subtree, workspace, template, target, caller instance, and limit.
- `GET /api/v1/permission-packages/applications/{id}/impact` now returns read-only application impact with created object status, capability manual-review rows, and rollback review steps.
- `GET /api/v1/permission-packages/applications/{id}/impact` 现在返回只读应用影响复盘，包括已创建对象状态、能力人工评审项和回滚评审步骤。
- Management MCP now exposes `list_permission_package_applications` for admin agents to review applied template versions, created assignment ids, capability ids, and data scopes.
- Permission package drafts now include a deterministic `policyGate` that allows direct apply for low-risk packages and requires approval for write, export, admin, high-risk, critical-risk, confidential, or restricted allowed capabilities.
- Permission package approval requests are now persisted in memory and PostgreSQL so approval-required drafts can be reviewed and applied with evidence.
- `POST /api/v1/permission-packages/approval-requests`, `GET /api/v1/permission-packages/approval-requests`, approve, and reject endpoints now provide the approval-request loop for permission packages.
- `POST /api/v1/permission-packages:apply` now accepts `approvalRequestId` and rejects pending, rejected, missing, or mismatched approval requests before writing permissions.
- Permission package approval requests now expire after 24 hours and are consumed transactionally by the first successful package application, so expired, already-used, or concurrently reused approvals cannot write permissions.
- Management MCP now exposes `create_permission_package_approval_request`, `list_permission_package_approval_requests`, `approve_permission_package_approval_request`, and `reject_permission_package_approval_request`.
- Web console AI Admin now shows the latest permission package application evidence after a successful apply.
- Web console AI Admin now includes a read-only application impact review panel under application evidence, with created/active/missing object counts, grant-object rows, capability manual-review rows, and rollback review steps in English and Simplified Chinese.
- Web 控制台 AI Admin 现在在应用证据下提供只读应用影响复盘，以中英双语展示已创建/有效/缺失对象数量、授权对象行、能力人工评审行和回滚评审步骤。
- Web console AI Admin now shows policy-gate status and disables direct apply when a package requires approval.
- Web console AI Admin now exposes the approval-required package path with create approval request, approve, reject, and approved apply controls in English and Simplified Chinese.
- Web console AI Admin now includes a Reviewer queue for routed pending approval requests, with reviewer-scoped refresh plus approve/reject actions in English and Simplified Chinese.
- Web 控制台 AI Admin 现在提供审查员队列，可按审查员刷新已路由的待处理审批，并以中英双语完成批准/驳回操作。
- Web console now includes a read-only effective permission explanation panel backed by `GET /api/v1/access-decisions:explain`, showing allow/deny outcome, evidence layers, data scopes, and next actions in English and Simplified Chinese.
- Web 控制台现在提供只读有效权限解释面板，由 `GET /api/v1/access-decisions:explain` 驱动，以中英双语展示允许/拒绝结果、证据层、数据范围和下一步动作。
- Web console AI Admin now includes a live approval journey workbench that creates a three-level tenant tree, discovers support tools, approves and applies a permission package, runs subject-scoped allow/deny MCP calls, and surfaces profile plus audit evidence.
- Web console AI Admin now shows first-run readiness for API, Mock MCP, subject-header CORS, private-upstream mode, and live data source before running the approval journey.
- Added `make scenario-permission-package-approval` to prove the approval-required permission package journey with local MCP discovery, approval, approved apply, runtime allow/deny, access-profile, application, and audit evidence.
- Added `make ai-admin-browser-journey` to start the local browser stack, verify subject-header CORS, and run the approval-required journey as a release-candidate gate.
- Added `make production-hardening` and a CI gate to verify admin-key enforcement, permission-package and management MCP protection, default private-upstream rejection, and public MCP endpoint registration before release-readiness claims.
- Permission package approval queues can now be filtered by configured reviewer routing rules, and approve/reject operations enforce reviewer tenant-subtree and workspace scope when `AGENT_HARBOR_APPROVAL_REVIEWERS` is configured.
- 权限包审批队列现在可按已配置审查员路由规则过滤；配置 `AGENT_HARBOR_APPROVAL_REVIEWERS` 后，批准/驳回会强制校验审查员的租户子树和工作区范围。

### Changed

- Reframed README, roadmap, and the v0.2.0 AI Admin journey note in English and Simplified Chinese around tenant-first access governance and AI-friendly permission operations instead of generic MCP gateway positioning.
- 将 README、路线图和 v0.2.0 AI Admin 用户旅程说明改为中英双语，并把产品叙事收敛到租户优先访问治理与 AI 友好权限运营，而不是通用 MCP Gateway 定位。

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
