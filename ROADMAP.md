# Roadmap

AgentHarbor is focused on tenant-first access governance and permission operations for AI agents, MCP tools, OpenAPI services, and governed data access.

AgentHarbor 聚焦于 AI Agent、MCP 工具、OpenAPI 服务和受治理数据访问的租户优先访问治理与权限运营。

This roadmap is intentionally high level. Detailed implementation work should still go through issues and pull requests.

本路线图保持高层级描述；具体实现仍应通过 issue 和 pull request 推进。

## Product Direction / 产品方向

AgentHarbor supports MCP gateway capabilities, but its primary product surface is not generic MCP aggregation. The core journey is permission operations: describe a tenant-scoped access need, draft a package, simulate the effective access result, route approval when risk requires it, apply through the existing grant chain, and inspect evidence afterward.

AgentHarbor 支持 MCP 网关能力，但主要产品界面不是通用 MCP 聚合。核心用户旅程是权限运营：描述一个租户范围的访问需求，生成权限包草案，模拟有效访问结果，在风险需要时进入审批路由，通过现有授权链落地，并在事后查看证据。

## Near Term / 近期

- Add approver roles and policy-configured approval routing for approval-required permission package application.
  增加审批角色和基于策略的审批路由，用于需要审批的权限包应用。
- Add approval review queues scoped by tenant subtree, workspace, template, target, caller instance, and status.
  增加按租户子树、工作区、模板、目标、调用方实例和状态过滤的审批队列。
- Add richer permission package recommendations and templates for non-technical administrators.
  为非技术管理员增加更丰富的权限包推荐和模板。
- Add effective permission explanations that show why a subject is allowed or denied across tenant, workspace, caller, capability, and data-scope layers.
  增加有效权限解释，展示主体在租户、工作区、调用方、能力和数据范围各层为什么被允许或拒绝。
- Expand permission package application health into richer diff and rollback review flows.
  将权限包应用健康巡检扩展为更丰富的差异和回滚评审流程。
- Harden tenant access profiles with clearer inherited-vs-direct permission views.
  强化租户访问画像，明确展示继承权限与直接权限。

## Next / 下一阶段

- Add approval notification hooks and review queues for high-risk permission package application.
  为高风险权限包应用增加审批通知 Hook 和评审队列。
- Add package version conflict remediation and data-scope repair flows before apply.
  在应用前增加权限包版本冲突修复和数据范围修复流程。
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
