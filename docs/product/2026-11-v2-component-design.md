# AgentHarbor v2 组件设计（2026-11-04）

> 状态: Component design，承接 [2026-11-v2-redesign.md](2026-11-v2-redesign.md) 的方向，把 M1/M2 的关键构件设计到可施工粒度。
> 范围: 策略文件格式、`harbor import` 行为、统一审计事件模型、`harbor-wrap` 架构、威胁模型边界。实现仍走 issue / PR。
> 语言: 设计阶段中文优先；合入主文档前按仓库惯例补齐中英双语。

## 1. 策略文件：`harbor.yaml`

单文件、git 友好、人类可 diff。它是策略的**唯一真源**；数据库只是 `harbor apply` 的物化状态。

```yaml
version: 1

workspaces:
  - id: main                     # 唯一可见的边界概念
    strict: false                # false=区内默认放行（高危标红）；true=默认拒
    targets:                     # 圈内可见的 MCP 目标
      - id: crm
        endpoint: https://crm.internal/mcp
    domains: [sales, billing]    # 数据域声明（严格模式下 data scope 用）

agents:
  - id: codex                    # 调用方，对应一把 agent key
    workspace: main
    allow:                       # 严格模式下=显式放行；宽松模式下=高危例外放行
      - capability: "crm.*"      # 整 server 放行是默认习惯用法：常用粒度在 server 级（MCPHub 观察）
      - capability: crm.lookupCustomer
    deny:                        # 两种模式下都生效：例外收紧
      - capability: "crm.*delete*"
        reason: 生产库只读
    subjects: ["user:support-*"] # 可代行的主体（可选，继承自 subject selector 语义）
```

判定语义（七条，全部可实现为纯函数）：

1. 匹配用 glob：`capability` 支持精确名与 `*` 通配；**整 server 放行（`crm.*`）是文档推荐的默认习惯用法**，逐工具枚举是可选深度。
2. **deny 永远赢**：命中 deny 即拒，无论模式。
3. 宽松模式（strict=false）：未命中 deny 即放行；高危调用放行但在审计事件上打风险标记。
4. 严格模式（strict=true）：命中 allow 才放行，其余拒；拒的原因引用规则位置（文件 + 行）。
5. 跨工作区：目标不在 agent 所属工作区的 `targets` 里 = 拒（与模式无关——边界是边界）。
6. 不可变默认：`version` 不认识的字段报错拒绝加载（防未来语法静默失效）。
7. **可见性即权限（严格模式）**：`tools/list` 只返回策略放行的工具——agent 根本看不到会被拒的工具，不浪费调用量、prompt 上下文干净（Stripe 模式：key 权限决定工具可见性，而非仅决定放行）。宽松模式不过滤，只标红。

工具链：`harbor apply --dry-run` 输出 diff；`harbor apply` 幂等物化（内部照旧展开三层链存储，表面无感）；文件可放 git，PR review 即审批。schema 校验进 `make check`。

## 2. `harbor import`

输入源（按优先级自动发现，可 `--from` 指定）：`~/.agent-shared/mcp/servers.json`（可移植目录）→ Claude Code 配置（`~/.claude.json` 的 mcpServers / 项目 `.mcp.json`）→ Codex 配置。四步行为：

1. **读 server 清单**，提议 targets（id/endpoint/channel）。上游凭据若在源配置的环境变量里，读取**值**迁入 harbor 服务端加密存储（internal/security 现成能力）；终端只显示掩码，`--confirm` 确认后才落库。
2. **预注册 agents**（codex/claude/kimi 等，按发现的客户端命名），打印一次性 agent key。
3. **生成初始 `harbor.yaml`**：单一 main 工作区、导入目标全进圈、`strict: false`——现状即基线，迁移零行为变化。
4. **`harbor connect <client>` 直接接线**：探测本机已安装的客户端（claude-code、cursor、codex…），保持 server 名不变（工具 ID 前缀不变，agent 无感）、端点换成 harbor、注入 agent key 引用，**直接写入该客户端的配置**；`--print` 退回仅打印片段。mcpm 式集中配置目录同格式可读——与配置同步工具互操作，不在纯同步上竞争（harbor 的增量永远是策略与审计）。

幂等：重复运行输出 diff（新发现的 server、已消失的标记 stale）。time-to-first-audit-event（目标 5 分钟）由此路径度量。全程不需要控制台：**无头模式（不启 Web 界面，仅 CLI + API + MCP 端点）是一等运行形态**，不是隐藏选项。

## 3. 统一审计事件

网关与 wrap 汇入同一模型（M2 起带 `source` 标记）：

```json
{
  "id": "evt_...", "ts": "...", "source": "gateway|wrap",
  "workspace": "main", "agent": "codex", "target": "crm",
  "capability": "crm.bulkDelete", "method": "tools/call",
  "decision": "allow", "reason": "default-open|rule:<file:line>|strict-default|boundary",
  "highRisk": true, "riskFlags": ["delete", "write", "export", "admin"],
  "subject": "user:support-amy", "latencyMs": 42,
  "outcome": "ok|error|denied", "argsDigest": "sha256:..."
}
```

要点：参数只存**摘要**（argsDigest，sha256），原文按数据域脱敏策略另存或丢弃——审计记录本身不得成为新的泄密面。`riskFlags` 是启发式标记，两处来源：Capability 模型现成的 `action` 字段（delete/write/export/admin）+ 名称模式（drop|delete|remove|truncate|purge|export|secret|credential）。标红只影响展示与检索（`harbor audit --high-risk`），不影响判定。拒绝事件的 `reason` 与错误消息同源——「拒绝即解释」的数据基础。

## 4. `harbor-wrap` 架构

```
agent 客户端 ←stdio JSON-RPC→ harbor-wrap ←stdio→ 真实 MCP server
                    │
                    ├─ 快照: harbor-policy.snapshot（`harbor sync` 拉取，本地文件）
                    └─ 暂存: ~/.harbor/spool/*.jsonl（断网累积，sync 时上送）
```

- **透传面**：initialize 握手、tools/list、notifications 原样透传；agent 对 wrap 无感知。
- **拦截面**：仅 `tools/call`。本地用快照按第 1 节六条语义判定；拒时返回标准 MCP 错误，错误结构携带 `{policy, reason, auditId}`（2026-07-28 错误结构装得下）——调用现场自解释，不必开控制台。
- **判定不联网**：调用路径零网络依赖（本地优先）；快照在启动时读取，SIGHUP 或 mtime 变化时热重载。
- **审计不丢**：事件先写本地 JSONL 暂存，`harbor sync`（或设 `HARBOR_URL` 后的惰性后台上送）负责上送；暂存文件 append-only，上送成功才截断。
- 单二进制，`harbor-wrap -- <server 启动命令>` 即全部用法。

## 5. 威胁模型与诚实边界

**防什么**：跨界访问与严格模式下的未授权调用（策略层）；事后无法追责（审计层）；客户端密钥散落（网关模式下凭据服务端加密，客户端零密钥）。

**不防什么（如实声明，写进 README）**：

1. 宽松模式下圈内为所欲为——这是默认全通的明码代价，缓解靠标红检索与 strict 开关，不靠假装。
2. 裸 shell 与非 MCP 工具（Replit 类事故的多数形态）；只能覆盖 MCP server 形态（含 filesystem server）。
3. 提示注入诱导的「合法但错误」调用——权限管边界，不管语义。
4. 绕过 wrap 直连真实 server 的本机进程（wrap 不是强制卡点，是自觉卡点；强制卡点=网关模式 + 网络层只允许网关出流量）。

## 6. 开放问题（不阻塞 M1）

- 快照签名：wrap 校验快照真伪（v1 仅 checksum；若多机分发再考虑签名）。
- 多工作区的文件组织：单文件多 workspace vs 每工作区一目录（先单文件，痛了再拆）。
- wrap 模式下的上游凭据：本地 server 仍需本地凭据，零密钥仅网关模式成立（README 同步声明）。
- per-agent 凭据绑定：一把 agent key 绑定个人上游凭据（MCPHub 的 per-user credentials 模式），隔离的 stdio 运行时随之而来。
- 工具侧沙箱：wrap 模式下被包装 server 的容器化隔离与最小主机权限（Docker MCP Gateway 模式）。
- 响应脱敏插件：工具返回值中的敏感信息掩码（Lasso token masking 模式）；请求侧已有 argsDigest，响应侧暂缺。

## 7. 设计出处（2026-11-04 同类产品调研）

| 设计 | 出处 |
| --- | --- |
| 可见性即权限（严格模式过滤 `tools/list`） | [Stripe MCP](https://docs.stripe.com/mcp)：restricted key 权限决定工具可见性；2026-12 起拒收全权限 key |
| `harbor connect` 单命令接线 | [Docker MCP Gateway](https://github.com/docker/mcp-gateway)：`--connect cursor`；profile 可经 OCI registry 分享（工作区可移植性的未来参照） |
| 整 server 粒度为默认习惯用法 | [MCPHub](https://github.com/samanhappy/mcphub)（2.6k star，同生态位最强对照）：仅 server/分组可见性 + 认证，无 per-tool 策略——粒度常在 server 级 |
| 无头模式一等化；文件存储默认 + PG 可选 | MCPHub：`DISABLE_WEB` 显式 flag、Database Mode 可选 |
| 零迁移注册（`wrap --register` 候选）与响应脱敏 | [Lasso mcp-gateway](https://github.com/lasso-security/mcp-gateway)：读现有客户端配置自注册、护栏插件化 |
| 配置互操作不竞争 | [mcpm](https://github.com/pathintegral-institute/mcpm.sh)：集中配置同步已有专职工具，harbor 读同格式作输入 |

差异化对照（调研确认的空位）：以上产品无一同时具备「工具级策略文件 + 跨全部 server 的证据链审计 + 拒绝即解释」。
