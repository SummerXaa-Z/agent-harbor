# Changelog

All notable public changes to AgentHarbor will be documented in this file.

This project uses Keep a Changelog-style sections and semantic versioning for tagged releases.

## [Unreleased]

### Added

- Web console Permission Packages now starts with a compact five-step go-live rail across change request, approval, permission application, runtime verification, and production readiness, with detailed acceptance evidence and permission configuration collapsed behind the summary.
- Web 控制台权限包页现在首屏展示紧凑的五步上线状态轨，按变更申请、审批、权限落地、运行验证和上线验收汇总状态，并把详细验收证据和权限配置收敛到摘要之后。
- Web console Permission Packages now treats not-yet-requested and pending approvals as pending work instead of production blockers, adds an approval entry CTA to the product message, and keeps the tenant access profile in the dedicated Access workspace.
- Web 控制台权限包页现在把未发起和待处理审批呈现为待处理事项而非生产阻断，在产品主张区提供发起权限审批入口，并将租户访问画像保留在独立的权限工作区。
- The web console now defaults to Permission Packages as the production approval and acceptance workspace, while the former Cockpit is renamed Self-Check for lower-level core permission loop validation.
- Web 控制台现在默认进入权限包审批与验收工作台；原 Cockpit 降级并重命名为自检，用于底层核心权限链路验证。
- Permission package templates now expose a stable version.
- Permission package application records are persisted in memory and PostgreSQL and returned from `POST /api/v1/permission-packages:apply`.
- `GET /api/v1/permission-packages/applications` lists permission package applications by tenant subtree, workspace, template, target, caller instance, and limit.
- `GET /api/v1/permission-packages/applications/health` now returns read-only ready, drifted, and needs-review health summaries derived from each application's impact review.
- `GET /api/v1/permission-packages/applications/health` 现在返回只读落地状态汇总，根据影响复核派生正常、已漂移和需复核状态。
- `GET /api/v1/permission-packages/applications/{id}/impact` now returns read-only application impact with created object status, capability manual-review rows, and rollback review steps.
- `GET /api/v1/permission-packages/applications/{id}/impact` 现在返回只读影响复核，包括已创建对象状态、能力人工评审项和回滚评审步骤。
- `GET /api/v1/permission-packages/applications/{id}/impact` now also returns a read-only remediation plan with ordered manual-review, drift-investigation, grant-disable, and final verification actions.
- `GET /api/v1/permission-packages/applications/{id}/impact` 现在还返回只读处置计划，包括按顺序排列的人工评审、漂移调查、授权禁用和最终校验动作。
- `GET /api/v1/permission-packages/applications/{id}/impact` now includes stable rollback and remediation `blockerCodes` for missing grant objects, inactive grant objects, and applications without recorded allowed capabilities.
- `GET /api/v1/permission-packages/applications/{id}/impact` 现在为缺失授权对象、未启用授权对象、无已记录允许能力的应用返回稳定的回滚与处置 `blockerCodes`。
- `GET /api/v1/permission-packages/applications/{id}/impact?rehearsal=grant_drift` now returns a response-only drift rehearsal with rehearsal metadata, simulated missing/inactive grant blockers, and read-only remediation actions without mutating permission state.
- `GET /api/v1/permission-packages/applications/{id}/impact?rehearsal=grant_drift` 现在返回仅影响响应的漂移演练，包含演练元数据、模拟的缺失/未启用授权阻断和只读处置动作，不会写入权限状态。
- `POST /api/v1/permission-packages:preflight` now returns read-only apply preflight with blockers, warnings, planned grant objects, approval readiness, data-scope fit, and existing grant-chain evidence before permission writes.
- `POST /api/v1/permission-packages:preflight` 现在返回只读应用前预检，在写入权限前展示阻断项、风险提示、计划授权对象、审批就绪状态、数据范围匹配和已有授权链证据。
- `GET /api/v1/permission-packages/production-readiness` now returns a read-only production go/no-go gate that combines preflight, latest application, health, impact, access-profile, runtime trace, and applied audit evidence.
- `GET /api/v1/permission-packages/production-readiness` 现在返回只读上线门禁，汇总预检、最新应用、落地状态、影响复核、访问画像、运行追踪和应用审计证据。
- `GET /api/v1/permission-packages/production-readiness/report` now exports a bounded read-only JSON evidence report for the production readiness decision.
- `GET /api/v1/permission-packages/production-readiness/report` 现在导出有边界的只读 JSON 证据报告，用于说明上线验收判断。
- Management MCP now exposes `list_permission_package_applications` for admin agents to review applied template versions, created assignment ids, capability ids, and data scopes.
- Management MCP now exposes `preflight_permission_package` so admin agents can verify permission package safety before calling `apply_permission_package`.
- Management MCP 现在提供 `preflight_permission_package`，管理 Agent 可以在调用 `apply_permission_package` 前验证权限包安全性。
- Management MCP now exposes `check_permission_package_production_readiness` so admin agents can ask for the same production go/no-go result without calling each evidence endpoint separately.
- Management MCP 现在提供 `check_permission_package_production_readiness`，管理 Agent 可以直接获取同一套生产上线判断，而不必分别调用每个证据端点。
- Management MCP now exposes `export_permission_package_production_evidence` so admin agents can produce the same bounded evidence report for handoff.
- Management MCP 现在提供 `export_permission_package_production_evidence`，管理 Agent 可以生成同一份有边界的证据报告用于交接。
- Permission package drafts now include a deterministic `policyGate` that allows direct apply for low-risk packages and requires approval for write, export, admin, high-risk, critical-risk, confidential, or restricted allowed capabilities.
- Permission package approval requests are now persisted in memory and PostgreSQL so approval-required drafts can be reviewed and applied with evidence.
- `POST /api/v1/permission-packages/approval-requests`, `GET /api/v1/permission-packages/approval-requests`, approve, and reject endpoints now provide the approval-request loop for permission packages.
- `POST /api/v1/permission-packages:apply` now accepts `approvalRequestId` and rejects pending, rejected, missing, or mismatched approval requests before writing permissions.
- Permission package approval requests now expire after 24 hours and are consumed transactionally by the first successful package application, so expired, already-used, or concurrently reused approvals cannot write permissions.
- Management MCP now exposes `create_permission_package_approval_request`, `list_permission_package_approval_requests`, `approve_permission_package_approval_request`, and `reject_permission_package_approval_request`.
- Web console Permission Packages now shows the latest permission package application evidence after a successful apply.
- Web console Permission Packages now includes a read-only application health panel with healthy, drifted, and review-needed rows plus one-click impact review in English and Simplified Chinese.
- Web 控制台权限包页现在提供只读落地状态面板，以中英双语展示正常、已漂移和需复核应用，并可一键进入影响复核。
- Web console Permission Packages now includes a read-only application impact review panel under application evidence, with created/active/missing object counts, grant-object rows, capability manual-review rows, and rollback review steps in English and Simplified Chinese.
- Web 控制台权限包页现在在应用证据下提供只读影响复核，以中英双语展示已创建/有效/缺失对象数量、授权对象行、能力人工评审行和回滚评审步骤。
- Web console Permission Packages now renders the read-only remediation plan for a permission package application in English and Simplified Chinese, without adding rollback execution controls.
- Web 控制台权限包页现在以中英双语展示权限包应用的只读处置计划，且不提供回滚执行控件。
- Web console Permission Packages now localizes rollback and remediation blockers from stable blocker codes instead of exposing raw backend text.
- Web 控制台权限包页现在通过稳定 blocker code 本地化回滚与处置阻塞原因，不再直接暴露后端原始文本。
- Web console Permission Packages now includes a read-only **Rehearse drift** action beside impact review so operators can preview drift blockers and then switch back to the real ready impact.
- Web 控制台权限包页现在在影响复核旁提供只读 **演练漂移** 动作，操作员可预览漂移阻断并切回真实 ready 影响。
- Web console Permission Packages now includes an Apply Preflight panel and blocks live apply when preflight reports blockers.
- Web 控制台权限包页现在提供应用前预检面板，并在预检返回阻断项时阻止实时应用。
- Web console Permission Packages now includes a Production Readiness panel in English and Simplified Chinese, showing ready, review-needed, or blocked status with runtime and audit evidence.
- Web 控制台权限包页现在提供中英双语上线验收面板，展示可上线、需复核或阻断状态，并列出运行和审计证据。
- Web console Permission Packages can export the read-only production evidence report as local JSON from the Production Readiness panel.
- Web 控制台权限包页现在可以在上线验收面板中把只读上线证据报告导出为本地 JSON。
- Web console Permission Packages now shows policy-gate status and disables direct apply when a package requires approval.
- Web console Permission Packages now exposes the approval-required package path with create approval request, approve, reject, and approved apply controls in English and Simplified Chinese.
- Web console Permission Packages now includes a Reviewer queue for routed pending approval requests, with reviewer-scoped refresh plus approve/reject actions in English and Simplified Chinese.
- Web 控制台权限包页现在提供审查员队列，可按审查员刷新已路由的待处理审批，并以中英双语完成批准/驳回操作。
- Web console now includes a read-only effective permission explanation panel backed by `GET /api/v1/access-decisions:explain`, showing allow/deny outcome, evidence layers, data scopes, and next actions in English and Simplified Chinese.
- Web 控制台现在提供只读权限判定说明面板，由 `GET /api/v1/access-decisions:explain` 驱动，以中英双语展示允许/拒绝结果、证据层、数据范围和下一步动作。
- Web console Permission Packages now includes a production acceptance flow that creates a three-level tenant tree, discovers support tools, approves and applies a permission package, runs subject-scoped allow/deny MCP calls, and surfaces profile plus audit evidence.
- Web console Permission Packages now shows runtime checks for API, Mock MCP, subject-header CORS, private-upstream mode, and live data source before running production acceptance.
- Added `make scenario-permission-package-approval` to prove the approval-required permission package journey with local MCP discovery, apply preflight, approval, approved apply, runtime allow/deny, access-profile, application, audit, production readiness, and production evidence report checks.
- 新增 `make scenario-permission-package-approval`，用于验证需要审批的权限包旅程，包括本地 MCP 发现、应用前预检、审批、已审批应用、运行时允许/拒绝、访问画像、应用记录、审计、上线验收和上线证据报告检查。
- Added `make ai-admin-browser-journey` to start the local browser stack, verify subject-header CORS, and run the approval-required journey as a release-candidate gate.
- `make ai-admin-browser-journey` now passes its isolated frontend origin into the API CORS allow-list, so release-candidate browser gates can run on non-default local ports while still verifying `X-AgentHarbor-Subject-Id`.
- `make ai-admin-browser-journey` 现在会把隔离前端 origin 传入 API CORS 白名单，因此发布候选浏览器门禁可以在非默认本地端口运行，并继续验证 `X-AgentHarbor-Subject-Id`。
- Added `make production-hardening` and a CI gate to verify admin-key enforcement, permission-package and management MCP protection, default private-upstream rejection, and public MCP endpoint registration before release-readiness claims.
- Permission package approval queues can now be filtered by configured reviewer routing rules, and approve/reject operations enforce reviewer tenant-subtree and workspace scope when `AGENT_HARBOR_APPROVAL_REVIEWERS` is configured.
- 权限包审批队列现在可按已配置审查员路由规则过滤；配置 `AGENT_HARBOR_APPROVAL_REVIEWERS` 后，批准/驳回会强制校验审查员的租户子树和工作区范围。

### Changed

- Reframed README, roadmap, and the v0.2.0 Permission Packages journey note in English and Simplified Chinese around tenant-first access governance and AI-friendly permission operations instead of generic MCP gateway positioning.
- 将 README、路线图和 v0.2.0 权限包用户旅程说明改为中英双语，并把产品叙事收敛到租户优先访问治理与 AI 友好权限运营，而不是通用 MCP Gateway 定位。

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
