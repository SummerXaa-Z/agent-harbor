# Roadmap

AgentHarbor is focused on tenant-first access governance and permission operations for AI agents, MCP tools, OpenAPI services, and governed data access.

AgentHarbor 聚焦于 AI Agent、MCP 工具、OpenAPI 服务和受治理数据访问的租户优先访问治理与权限运营。

This roadmap is intentionally high level. Detailed implementation work should still go through issues and pull requests.

本路线图保持高层级描述；具体实现仍应通过 issue 和 pull request 推进。

## Product Direction / 产品方向

AgentHarbor supports MCP gateway capabilities, but its primary product surface is not generic MCP aggregation. The core journey is permission operations: describe a tenant-scoped access need, draft a package, simulate the effective access result, route approval when risk requires it, apply through the existing grant chain, and inspect evidence afterward.

AgentHarbor 支持 MCP 网关能力，但主要产品界面不是通用 MCP 聚合。核心用户旅程是权限运营：描述一个租户范围的访问需求，生成权限包草案，模拟有效访问结果，在风险需要时进入审批路由，通过现有授权链落地，并在事后查看记录。

## Current: v0.6.0 / 当前版本：v0.6.0

AgentHarbor is at v0.6.0 and still scoped to local evaluation, design feedback, early integration, and pilot deployments behind the documented production preflight. It is not recommended for production traffic without that preflight.

AgentHarbor 当前为 v0.6.0，范围仍是本地评估、设计反馈、早期集成，以及通过文档化生产预检的试点部署；未经该预检不建议承载生产流量。

v0.4.0 replaced the console with the redesigned dual-surface experience; v0.5.0–v0.5.3 added the notification center phases, approval-detail dual columns, absolutized copied client configuration, and auditable access-handoff config preview/copy; v0.6.0 adds per-holder views. This does not change the developer-preview positioning.

v0.4.0 起控制台切换为重设计的双面体验；v0.5.0–v0.5.3 增加了通知中心各阶段、审批明细双列、复制配置完整地址与可审计的交接配置预览/复制；v0.6.0 增加持有者视图。这不会改变当前开发者预览定位。

- The redesigned dual-surface console pairs a user workbench (access query, permission requests, my permissions, go-live handoff) with a management console (approvals, capability governance, registry, tenants, traces), plus a command palette, notification center, and mobile-ready layouts.
  重设计双面控制台将用户工作台（访问查询、申请权限、我的权限、上线交接）与管理控制台（变更审批、能力治理、资源管理、租户组织、运行审计）配对，并提供命令面板、通知中心与移动端可用布局。
- Per-holder views let platform administrators bind managed admin identities to the caller agents they operate, narrowing that identity's user surface server-side; empty bindings keep tenant-scope visibility.
  持有者视图允许平台管理员把托管管理员身份绑定到其负责的调用方，在服务端收窄该身份的用户面；空绑定保持租户范围可见性。

- Permission Changes supports deterministic package drafts, allow/deny simulation, policy gates, approval-required apply, read-only preflight, application health, impact review, go-live status, and bounded acceptance-report export.
  权限变更已支持确定性权限包草案、允许/拒绝模拟、策略门禁、需审批应用、只读预检、落地状态、影响复核、上线状态和有边界的验收报告导出。
- Tenant-first governance covers tenant, workspace, caller, capability, and data-scope enforcement, with scoped administrators, managed administrator identities, tenant permission center views, and audit records.
  租户优先治理已覆盖租户、工作区、调用方、能力和数据范围控制，并具备范围化管理员、托管管理员身份、租户权限中心视图和审计记录。
- Management MCP exposes permission-operation tools with safety, access, lifecycle, execution, and confirmation metadata so admin-agent clients can inspect boundaries before writes.
  Management MCP 已暴露带安全、访问、生命周期、执行和确认元数据的权限运营工具，便于管理 Agent 在写入前检查边界。
- Access Handoff extends a ready permission application into copyable MCP configuration, prompt guidance, explicit permission boundaries, and administrator-issued one-time short-lived tokens with revocation and audit references.
  接入交付把已就绪的权限应用延伸为可复制的 MCP 配置、提示词指引、明确的权限边界，以及由管理员签发、一次展示、可撤销且带审计引用的短期 Token。
- Access Query can preserve one exact requested capability through approval, application, readiness, handoff, and governed runtime use without silently expanding to a template bundle.
  访问查询可以把一项精确申请能力持续传递到审批、应用、上线状态、接入交付和受治理运行时，不会静默扩展为整个权限包。
- Local validation is anchored by `make check`, `make release-check`, `make evaluation-readiness`, PR CI, and main-branch CI.
  本地验收以 `make check`、`make release-check`、`make evaluation-readiness`、PR CI 和 main 分支 CI 为准。

## Near Term / 近期

- Keep running the external evaluator loop on released baselines (rounds 4–7 are archived in `docs/product/0.4.0-console-eval.md`), with `time-to-first-report` and first-blocker records as the primary inputs; complete the round-7 deferrals (EN spot-check and free exploration) in the next round.
  持续在已发布基线上运行外部评估闭环（第 4–7 轮已归档于 `docs/product/0.4.0-console-eval.md`），以 `time-to-first-report` 和首个阻塞点记录作为主要输入；下一轮补完第七轮推迟的英文抽查与自由探索。
- Close the round-7 follow-ups: resource lists occasionally rendering stale rows after write operations (reload currently restores them) and the cockpit environment-check coverage wording.
  收敛第七轮后续项：写操作后资源列表偶发陈旧（当前靠刷新恢复）与驾驶舱环境检查口径文案。
- Fix repeated evaluator blockers before adding new product surface area.
  新增产品界面前，先修复外部评估中重复出现的阻塞点。
- Keep release-candidate hardening limited to setup reliability, Permission Changes comprehension, report trust, security regressions, and documentation gaps.
  发布候选加固只覆盖启动可靠性、权限变更可理解性、报告可信度、安全回归和文档缺口。

## Next / 下一阶段

- Refine the managed-identity and holder-binding model from operational feedback.
  根据运营反馈完善托管身份与持有者绑定模型。
- Add package version conflict remediation and data-scope repair flows before apply when evaluator feedback shows these block real usage.
  当外部评估显示版本冲突或数据范围修复阻碍真实使用时，再补应用前修复流程。
- Add OpenAPI capability discovery and assignment semantics alongside MCP tools.
  在 MCP 工具之外增加 OpenAPI 能力发现和分配语义。
- Add first-class data-system targets for data lakes, warehouses, and databases.
  增加面向数据湖、数据仓库和数据库的一等数据系统目标。
- Expand data-scope validation so administrators can catch invalid narrowing before runtime.
  扩展数据范围校验，让管理员能在运行前发现无效收敛。
- Improve audit exports and trace filtering for security review workflows.
  改进审计导出和 trace 过滤，服务安全评审流程。

## Later / 远期

- Add identity-provider integration for management-console operators.
  增加管理控制台操作员的身份提供商集成。
- Add policy simulation before publishing tenant, workspace, or caller-instance changes.
  在发布租户、工作区或调用方实例变更前增加策略模拟。
- Add observability integrations for metrics, traces, and structured audit sinks.
  增加指标、trace 和结构化审计接收端的可观测集成。
- Define versioned API compatibility guarantees as external integrations grow.
  随着外部集成增长，定义版本化 API 兼容性承诺。

## Non-Goals For The First Public Release / 首个公开版本非目标

- Replacing a full IAM system.
  不替代完整 IAM 系统。
- Competing as a generic MCP gateway or MCP server marketplace.
  不以通用 MCP Gateway 或 MCP Server 市场作为首版竞争目标。
- Granting unrestricted access to private-network upstream targets.
  不授予对私有网络上游目标的无限制访问。
- Inferring every possible tool argument schema without explicit capability metadata.
  不在缺少明确能力元数据时推断所有工具参数 schema。
- Supporting production multi-region deployment before the core permission model is stable.
  不在核心权限模型稳定前支持生产级多地域部署。
