# Roadmap

AgentHarbor is focused on tenant-first access governance and permission operations for AI agents, MCP tools, OpenAPI services, and governed data access.

AgentHarbor 聚焦于 AI Agent、MCP 工具、OpenAPI 服务和受治理数据访问的租户优先访问治理与权限运营。

This roadmap is intentionally high level. Detailed implementation work should still go through issues and pull requests.

本路线图保持高层级描述；具体实现仍应通过 issue 和 pull request 推进。

## Product Direction / 产品方向

AgentHarbor supports MCP gateway capabilities, but its primary product surface is not generic MCP aggregation. The core journey is permission operations: describe a tenant-scoped access need, draft a package, simulate the effective access result, route approval when risk requires it, apply through the existing grant chain, and inspect evidence afterward.

AgentHarbor 支持 MCP 网关能力，但主要产品界面不是通用 MCP 聚合。核心用户旅程是权限运营：描述一个租户范围的访问需求，生成权限包草案，模拟有效访问结果，在风险需要时进入审批路由，通过现有授权链落地，并在事后查看记录。

## Current Developer Preview / 当前开发者预览

The current v0.2 developer preview is scoped to local evaluation, design feedback, and early contribution. It is not recommended for production traffic.

当前 v0.2 开发者预览范围是本地评估、设计反馈和早期贡献；暂不建议承载生产流量。

v0.3 development is now underway with the first Access Handoff product slice. This does not change the production-readiness status of the developer preview.

v0.3 已进入开发阶段，首个产品切片是接入交付；这不会改变当前开发者预览尚未面向生产流量的定位。

- Permission Changes supports deterministic package drafts, allow/deny simulation, policy gates, approval-required apply, read-only preflight, application health, impact review, go-live status, and bounded acceptance-report export.
  权限变更已支持确定性权限包草案、允许/拒绝模拟、策略门禁、需审批应用、只读预检、落地状态、影响复核、上线状态和有边界的验收报告导出。
- Tenant-first governance covers tenant, workspace, caller, capability, and data-scope enforcement, with scoped administrators, managed administrator identities, tenant permission center views, and audit records.
  租户优先治理已覆盖租户、工作区、调用方、能力和数据范围控制，并具备范围化管理员、托管管理员身份、租户权限中心视图和审计记录。
- Management MCP exposes permission-operation tools with safety, access, lifecycle, execution, and confirmation metadata so admin-agent clients can inspect boundaries before writes.
  Management MCP 已暴露带安全、访问、生命周期、执行和确认元数据的权限运营工具，便于管理 Agent 在写入前检查边界。
- Access Handoff extends a ready permission application into copyable MCP configuration, prompt guidance, explicit permission boundaries, and administrator-issued one-time short-lived tokens with revocation and audit references.
  接入交付把已就绪的权限应用延伸为可复制的 MCP 配置、提示词指引、明确的权限边界，以及由管理员签发、一次展示、可撤销且带审计引用的短期 Token。
- Local validation is anchored by `make check`, `make release-check`, `make evaluation-readiness`, PR CI, and main-branch CI.
  本地验收以 `make check`、`make release-check`、`make evaluation-readiness`、PR CI 和 main 分支 CI 为准。

## Near Term / 近期

- Run the external evaluator loop with platform engineer, security reviewer, and tenant administrator roles, using `time-to-first-report` and first-blocker records as the primary inputs.
  用平台工程师、安全审核人和租户管理员三个角色跑外部评估，以 `time-to-first-report` 和首个阻塞点记录作为主要输入。
- Fix repeated evaluator blockers before adding new product surface area.
  新增产品界面前，先修复外部评估中重复出现的阻塞点。
- Keep release-candidate hardening limited to setup reliability, Permission Changes comprehension, report trust, security regressions, and documentation gaps.
  发布候选加固只覆盖启动可靠性、权限变更可理解性、报告可信度、安全回归和文档缺口。
- Prepare the v0.2 developer-preview tag and short release notes after local gates, PR CI, and main CI pass.
  本地门禁、PR CI 和 main CI 通过后，准备 v0.2 开发者预览标签和简短发布说明。

## Next / 下一阶段

- Stabilize Access Handoff through evaluator runs, browser review, PR CI, and a v0.3 release candidate before opening the My Access self-service slice.
  通过外部评估、浏览器复核、PR CI 和 v0.3 发布候选继续稳定接入交付，再开启 My Access 自助视图。
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
- Define versioned API compatibility guarantees after the first tagged release.
  在首个正式标签版本后定义版本化 API 兼容性承诺。

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
