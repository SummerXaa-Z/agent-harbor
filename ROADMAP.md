# Roadmap

AgentHarbor is the permission and audit layer for AI agent tool calls: every agent-to-tool call gets three answers — is it allowed, on what basis, and how to look it up afterwards.

AgentHarbor 是 AI Agent 工具调用的权限与审计层：每个 agent 对工具的调用都要回答三个问题——允许吗、依据什么、事后怎么查。

This roadmap was redirected on 2026-11-04. The reasoning, diagnosis, and asset accounting live in [docs/product/2026-11-v2-redesign.md](docs/product/2026-11-v2-redesign.md); this file carries the direction and the milestones. Implementation work still goes through issues and pull requests.

本路线图于 2026-11-04 重定向。理由、诊断与资产对账见 [docs/product/2026-11-v2-redesign.md](docs/product/2026-11-v2-redesign.md)；本文件承载方向与里程碑。具体实现仍通过 issue 和 pull request 推进。

## Why The Change / 为什么改

Through v0.6.2 AgentHarbor was shaped as a self-hosted multi-tenant governance platform, and the honest read is that the shape is wrong for the problem:

到 v0.6.2 为止，AgentHarbor 的形态是自托管多租户治理平台；诚实的结论是这个形态与问题不匹配：

- A network middleman cannot see stdio MCP servers, which is a large fraction of where agents actually connect, and the ecosystem keeps moving enforcement to the clients and servers, not the middle.
  网络中间人看不见 stdio MCP server——那占 agent 实际连接的很大一部分；而生态的执行点持续向客户端与服务端两端移动，不在中间。
- The governance console became the product, but no real team lives in a permission-governance console: small teams approve in PRs and chat, enterprises use their existing IAM and ticketing.
  治理控制台成了产品本身，但没有真实团队住在权限治理控制台里：小团队在 PR 和聊天工具里审批，企业用已有的 IAM 与工单系统。
- The multi-tenant platform data model fits a platform vendor, not a self-hosting team.
  多租户平台数据模型适合平台厂商，不适合自托管团队。
- Engineering quality is no longer the bottleneck — adoption is. Further internal polish has near-zero marginal return.
  工程质量已不是瓶颈，采用才是；继续内部打磨的边际收益接近零。

## Product Direction v2 / 产品方向 v2

One core, three delivery forms:

一个内核，三种交付形态：

- `harbor` (server, existing binary, narrowed): HTTP MCP gateway plus the audit store. The console keeps exactly two first-class surfaces — the audit viewer ("what did my agents do") and the decision explainer ("why was this denied").
  `harbor`（server，现有二进制，收缩范围）：HTTP MCP 网关与审计存储。控制台只保留两个一等公民——审计查看器（我的 agent 做了什么）与决策解释器（为什么被拒）。
- `harbor-wrap` (new): a stdio MCP wrapper. `harbor-wrap -- <server command>` tees every tool call into the audit model and enforces locally against a cached policy snapshot.
  `harbor-wrap`（新增）：stdio MCP 包装器。`harbor-wrap -- <server 命令>` 把每次工具调用记录进审计模型，并用缓存的策略快照在本地判定。
- `harbor-policy` (extracted): the decision engine as a standalone module — a request in, an effect plus evidence out. The server and the wrapper share it; we do not bet on third-party adoption.
  `harbor-policy`（抽取）：判定引擎独立成模块——请求进，判定加证据出。server 与 wrapper 共用；不以第三方采用为前提。

Design principles: single tenant by default (hierarchy hidden behind a flag, model untouched); configuration as files (YAML policies, PR review as the approval flow, the console approval engine becomes optional); enforcement follows the ecosystem (gateway for HTTP targets, wrapper for stdio targets); audit and explanation get the remaining console investment; freeze rather than delete the governance journeys.

设计原则：默认单租户（层级隐藏在 flag 之后，数据模型不动）；配置即文件（YAML 策略、PR review 即审批、控制台审批引擎降级为可选）；执行点跟随生态（HTTP 目标走网关、stdio 目标走 wrapper）；剩余的控制台投入全部进审计与解释；治理旅程冻结而非删除。

Target users: ourselves first (heavy multi-agent daily usage), then small self-hosting teams whose agents touch real tools. Not enterprise platform buyers.

目标用户：先是我们自己（重度多 agent 日常使用），随后是让 agent 接触真实工具的小型自托管团队。不是企业平台买家。

## M1 — Self-Host First / 自己先用起来（target v0.7.0, ~2 part-time weeks）

Scope:

- First boot auto-provisions a default tenant and workspace; the tenant hierarchy disappears from the UI unless multi-tenant mode is enabled.
  首启动自动建默认租户与工作区；除非启用多租户模式，租户层级从界面消失。
- The grant surface collapses to a single object — agent, capability, constraints, effect. The three-layer chain remains as internal storage and per-layer audit records, and disappears from every API and UI surface.
  授权表面塌缩为单一对象——agent、能力、约束、effect。三层链保留为内部存储与逐层审计账本，从一切 API 与界面表面消失。
- Inside a workspace the default is allow: capabilities are callable without explicit grants, high-risk calls execute but are flagged in the audit timeline, cross-workspace access always requires an explicit grant, and a per-workspace strict flag flips the default to deny, where single-line grants then apply.
  工作区内默认放行：能力无需显式授权即可调用，高风险调用照常执行但在审计时间线上标红，跨工作区访问永远需要显式授权；工作区级 strict 开关翻开即默认拒，此时单行 grant 生效。
- Policy configuration as files: YAML import for entitlements and route policies (export already exists in the management MCP), applied via CLI or file watch; PR review replaces the console approval flow as the default path.
  策略配置即文件：entitlement 与 route policy 的 YAML 导入（导出已存在于 management MCP），经 CLI 或文件监听生效；PR review 取代控制台审批流成为默认路径。
- `harbor import`: reads existing client MCP configuration (Claude Code and friends, or portable catalogs such as `~/.agent-shared/mcp/servers.json`), registers agents, proposes starter policies, and emits per-client config snippets pointing at harbor — no console required for bootstrap.
  `harbor import`：读取既有客户端 MCP 配置（Claude Code 等，或诸如 `~/.agent-shared/mcp/servers.json` 的可移植目录），注册 agent、给出起始策略建议、生成指向 harbor 的各客户端配置片段——引导全程不需要控制台。
- Console investment goes to the audit viewer and the decision explainer only; all other governance views freeze at their current state.
  控制台投入只进审计查看器与决策解释器；其余治理视图冻结在当前状态。

Acceptance: at least two of our own agent clients route HTTP MCP traffic through harbor for a full week; the audit page answers "what did my agents do yesterday" in under a minute; at least one policy denies a call and the explainer shows the evidence chain; a fresh install reaches its first audited tool call in under five minutes via the `harbor import` path (time-to-first-audit-event); a workspace flipped to strict mode denies an ungranted capability call, and flipping it back restores access.

验收：自己的至少两个 agent 客户端的 HTTP MCP 流量过 harbor 连续一周；审计页 1 分钟内回答「我的 agent 昨天做了什么」；至少一条策略实际拒绝一次调用且解释器展示证据链；全新安装经 `harbor import` 路径在五分钟内出现第一条被审计的工具调用（time-to-first-audit-event）；工作区翻开严格模式后未授权的能力调用被拒，翻回即恢复。

## M2 — Cover The stdio Half / 覆盖 stdio 半壁（target v0.8.0, ~3 part-time weeks）

Scope:

- The new data plane speaks the MCP 2026-07-28 stateless profile (SEP-2575) as its primary protocol: method and tool name travel in HTTP headers, capability-level authorization reads the `Mcp-Name` header, and the gateway carries no session affinity; session-based clients and targets are bridged during the transition.
  新数据面以 MCP 2026-07-28 stateless 规范（SEP-2575）为主协议：方法与工具名走 HTTP 头，能力级判定直接读 `Mcp-Name` 头，网关无会话粘性；过渡期内桥接存量会话式客户端与目标。
- `harbor-wrap`: a single-binary stdio MCP wrapper — pass-through proxy for stdio servers, tool calls teed to the audit model, enforcement from an embedded policy snapshot, local spool when the server is unreachable.
  `harbor-wrap`：单二进制 stdio MCP 包装器——stdio server 的透传代理、工具调用记录进审计模型、内嵌策略快照判定、server 不可达时本地暂存。
- Denials are self-explaining: the deny response carries the policy name, reason, and audit id in the MCP error structure; `harbor why <trace-id>` answers follow-ups from the CLI without opening the console.
  拒绝即解释：deny 响应在 MCP 错误结构中携带策略名、原因与 audit id；`harbor why <trace-id>` 在 CLI 里回答追问，不必打开控制台。
- SQLite as the default store (pure-Go driver) so one binary carries a durable audit log; PostgreSQL stays for scale.
  SQLite 作为默认存储（纯 Go 驱动），单个二进制即带可持久化的审计日志；PostgreSQL 保留给规模化场景。
- A unified audit timeline across gateway and wrapper sources (source tagging on events), re-centered on the "my agents" perspective — organized by agent and time rather than tenant hierarchy.
  网关与 wrapper 来源统一的审计时间线（事件带来源标记），视角换轴为「我的 agent」——按 agent 与时间组织，而非租户层级。

Acceptance: a stdio server (for example the filesystem MCP) runs wrapped end to end; the same policy produces the same decisions as on the gateway; wrapper-sourced events appear in the same audit timeline; an offline period loses no events; a denied call arrives with its policy name, reason, and audit id in the error payload.

验收：一个 stdio server（例如 filesystem MCP）完整跑在 wrapper 下；同一策略在网关与 wrapper 上产生相同判定；wrapper 来源事件出现在同一审计时间线；离线时段不丢事件；被拒的调用在错误负载中自带策略名、原因与 audit id。

## M3 — Extract The Engine / 抽离内核（target v0.9.0, ~2 part-time weeks）

Scope:

- The decision logic (effective scopes, subject selectors, evidence assembly) moves out of `internal/domain` into a standalone `harbor-policy` module with no store or console dependencies; existing fuzz targets move with it.
  判定逻辑（effective scopes、subject selector、证据组装）从 `internal/domain` 移入独立的 `harbor-policy` 模块，不依赖 store 与 console；现有 fuzz 目标随迁。
- A versioned policy snapshot format the wrapper can cache; a cross-implementation equivalence test asserts the same snapshot yields identical decisions on server and wrapper.
  版本化的策略快照格式供 wrapper 缓存；跨实现等价测试断言同一快照在 server 与 wrapper 上判定一致。

Acceptance: the equivalence test passes over the policy corpus; the policy module imports nothing from the server.

验收：等价测试在策略语料上通过；policy 模块不 import server 的任何东西。

## M4 — The Second User / 第二个用户（v1.0.0 candidate, ~1 week plus open-ended feedback）

Scope: quickstart deployment (docker-compose or package), README repositioned to the v2 story, `harbor init` bootstrap, and recruiting one to three real teams from our own network to run it. What v1.0.0 contains is decided by their feedback, not by this document.

范围：快速启动部署（docker-compose 或软件包）、README 按 v2 重写、`harbor init` 引导，并从自己的网络找一到三个真实团队使用。v1.0.0 的内容由他们的反馈决定，不由本文档决定。

Acceptance: one external team runs AgentHarbor in a real workflow and sends unprompted feedback.

验收：至少一个外部团队在真实流程中运行 AgentHarbor 并主动给出反馈。

## Stopped / 停止

- New external evaluation rounds (round 8 stays skipped).
  新的外部评估轮（round-8 维持跳过）。
- New governance concepts and console journeys; the apply–approve–go-live journeys and the permission-package state machine are frozen, not removed.
  新的治理概念与控制台旅程；apply–approve–go-live 旅程与权限包状态机冻结而非移除。
- Enterprise-facing features (SSO, SCIM, multi-tenant operations UI) until a real customer asks.
  企业向特性（SSO、SCIM、多租户运营界面），直到有真实客户提出。

## Non-Goals / 非目标

- Replacing a full IAM system. / 不替代完整 IAM 系统。
- Being a generic MCP gateway or MCP server marketplace. / 不做通用 MCP Gateway 或 MCP Server 市场。
- Enforcing on traffic the deployment cannot see; the wrapper exists precisely so we never pretend a network gateway covers stdio.
  不对部署形态看不见的流量假装执行；wrapper 的存在正是为了让网关不必假装覆盖 stdio。
- Granting unrestricted access to private-network upstream targets. / 不授予对私有网络上游目标的无限制访问。
- Versioned API compatibility guarantees before external integrations exist.
  在外部集成出现之前，不承诺版本化 API 兼容性。

## Prior Direction / 旧方向

Through v0.6.2 the roadmap targeted tenant-first permission operations with an evaluator-driven loop; the delivered feature set (dual-surface console, permission packages with approval routing, per-holder views, access handoff) is documented in the git history of this file, the CHANGELOG, and `docs/engineering/`. Those surfaces are frozen, not deleted.

到 v0.6.2 为止，路线图以租户优先的权限运营与评估闭环为目标；已交付的特性（双面控制台、带审批路由的权限包、持有者视图、接入交付）记录在本文件的 git 历史、CHANGELOG 与 `docs/engineering/` 中。这些界面冻结，但不会删除。
