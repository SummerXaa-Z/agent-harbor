# AgentHarbor v2 用户体验设计（2026-11-04）

> 状态: UX design，承接 [重设计](2026-11-v2-redesign.md) 第 6 节交互原则与 [组件设计](2026-11-v2-component-design.md)，把体验落成逐分钟旅程、命令面、文案规范与线框。
> 范围: 首次运行、日常 CLI、拒绝文案、周检仪式、控制台两页、边界体验。实现仍走 issue / PR。
> 语言: 设计阶段中文优先；合入主文档前按仓库惯例补齐中英双语。

## 1. 首次运行：五分钟电影（逐击键脚本）

目标：从 `harbor import` 到第一条被审计的调用 ≤ 5 分钟，全程不打开浏览器。

```text
$ harbor import
⟡ 扫描 MCP 配置…
  ✓ Claude Code    ~/.claude.json（4 个 server）
  ✓ Codex          ~/.codex/config.toml（2 个 server）
  ✓ 便携目录       ~/.agent-shared/mcp/servers.json（3 个，2 个与上面重复）

  发现 7 个目标：
    crm          https://crm.internal/mcp        3 tools
    filesystem   stdio: npx filesystem-mcp      11 tools
    …

  其中 2 个含上游凭据（CRM_API_KEY 等）。迁入 harbor 加密存储？[Y/n]
  （值不回显；迁移后提示从原配置删除）

$ harbor connect claude-code codex
  ✓ claude-code   已写入 ~/.claude.json（server 名不变，端点 → harbor）
  ✓ codex         已写入 ~/.codex/config.toml
  ⚠ 重启客户端后生效

  Agent keys（仅此一次展示，请妥善保存）：
    claude   ahx_9f3e…   已注入 claude-code 配置
    codex    ahx_71bc…   已注入 codex 配置

  harbor.yaml 已生成：工作区 main（宽松模式），7 个目标全部入圈。
⟡ 试试：harbor audit --follow
```

`harbor audit --follow` 随后实时流出事件——第一条审计出现即 onboarding 完成。空手起步（扫不到任何配置）时给出 `harbor init` 引导与手写目标示例，不报错退场。

## 2. CLI 命令面（全部词汇，每条一行）

| 命令 | 一句话 |
| --- | --- |
| `harbor init` | 生成空 harbor.yaml 骨架 |
| `harbor import` | 扫描客户端/目录配置 → 目标 + agent + 初始策略 |
| `harbor connect <client>` | 接线客户端（直接写配置；`--print` 兜底） |
| `harbor apply` | 校验并物化 harbor.yaml（`--dry-run` 出 diff） |
| `harbor audit` | 审计时间线：`--follow` 实时、`--high-risk` 仅红、`--json` 机器可读 |
| `harbor why <id>` | 单事件证据链长解释 |
| `harbor review` | 使用回顾与收紧建议：`--week`、`--suggest` |
| `harbor key issue/rotate/revoke` | agent key 生命周期 |
| `harbor sync` | wrap 侧：拉策略快照、上送暂存审计 |
| `harbor-wrap -- <cmd>` | stdio 包装器本体 |

终端规范：退出码可脚本化（0 成功 / 2 配置错误 / 3 策略拒绝 / 4 网络失败）；颜色可关（`--no-color`/非 TTY 自动）；一切输出有 `--json` 形态；人机文案 zh/en，**面向 LLM 的文案永远英文**（见下节）。

## 3. 拒绝文案：双读者设计（本节最重要）

拒绝消息的第一读者是 **LLM**（它在调用现场读到这段话），第二读者才是人。文案为 LLM 而写：

```text
DENIED · tool not in allow list (workspace "main" is strict)
policy: harbor.yaml:18 (allow: crm.*) · audit: evt_7f21c
Do not retry. Ask the operator if you need this capability.
```

规范五条：

1. 首行 = 判定 + 原因，≤10 个词——LLM 优先解析首行。
2. 永远携带 `policy` 引用（file:line）与 `audit` id。
3. **显式写 "Do not retry"**——不写这句，模型会反复重试。
4. 给出路（ask the operator），不给内部细节（无堆栈、无内部 id）。
5. 面向 LLM 的文案固定英文（与界面语言无关——语言切换只作用于人读的 UI）；`harbor why <audit-id>` 给人读的完整版（zh/en + 证据链逐步展开）。

宽松模式不拒调用，但高危调用的审计事件标红（见 §5），`harbor audit --high-risk` 一键过滤。

## 4. 周检仪式：`harbor review`（审计 → 收紧的闭环）

「审计教你知道该锁什么」的机械化——用真实用量产出策略建议：

```text
$ harbor review --week
⟡ 11-04 ~ 11-11 · 3 agents · 214 calls · 0 denied

  Top tools             calls
  crm.lookupCustomer      98
  crm.updateTicket        61
  filesystem.read         41
  crm.bulkDelete           8   ⚠ high-risk

  高危 8 次，全部来自 codex（crm.bulkDelete）。
  建议追加到 harbor.yaml（确认后 harbor apply）：
    deny:
      - capability: crm.bulkDelete    # 8 次调用，本周 0 次对应人工工单
```

建议引擎三条规则：高危高频调用 → deny 候选；严格模式下长期未命中的 allow 规则 → 清理候选；跨界尝试 → 直接报告。`--suggest` 只输出 diff，不自动改文件——PR review 仍是唯一改动通道。

## 5. 控制台两页（线框）

审计时间线（默认页，视角 = 「我的 agents」）：

```text
┌ 我的 agents ────────────────────────────────────────────────┐
│ [全部 agent ▾] [今天 ▾] [仅高危 ◉]        🔍 搜索工具/agent    │
│──────────────────────────────────────────────────────────────│
│ 09:41  codex → crm.lookupCustomer   allow           42ms     │
│ 09:41  codex → crm.bulkDelete    ⚠ allow·标红      880ms     │
│ 09:38  claude → filesystem.read      allow           12ms     │
│ 09:12  codex → crm.exportAll     ✖ denied   harbor.yaml:18  │
│        ↳ 证据链：边界检查 ✓ → deny 匹配 "crm.*delete*" → 拒    │
└──────────────────────────────────────────────────────────────┘
```

- 点开任意行 = 证据链抽屉（决策解释器）：判定走的每一步、命中的规则、`harbor why` 的网页版。
- 来源徽标（gateway / wrap）、参数摘要、延迟、agent 头像一行齐备。
- **v2 控制台只有这两页**（时间线 + 解释器）。冻结的治理视图默认不出现，多租户 flag 后面也不解锁给 v2 用户。

## 6. 边界与异常体验（规则表）

| 情形 | 行为 |
| --- | --- |
| harbor.yaml 非法 | `apply` 拒绝并报 file:line；**运行中的网关沿用上一份好策略**（坏文件不致瘫）；wrap 同理沿用上次好快照 |
| 严格模式 + 快照过期（>N 天未 sync） | wrap 拒绝放行并返回「policy stale, run harbor sync」——严格模式永不懈锁 |
| 宽松模式 + 快照过期 | 照常放行 + 标红（宽松模式的语义就是宽） |
| harbor 掉线 | 客户端表现为「该 MCP server 不可达」（与任何死 server 一致）；**wrap 不受影响**——本地优先是韧性卖点 |
| 暂存区写满 | 丢最旧事件 + 记一条「丢弃计数」事件——审计满永不禁掉工具调用 |
| key 被吊销 | 明确的 `DENIED · key revoked`，不是含糊 401 |
| import 一无所获 | 友好引导 `harbor init` + 手写目标示例，不报错退场 |

## 7. 设计出处与衔接

- 拒绝文案双读者：无先例可抄（同类产品均未认真设计 LLM 侧文案）——原创点，验证方式进 M1 验收（观察 agent 收到拒绝后的重试行为）。
- `harbor review` 的建议引擎是「渐进加固」叙事的机械化（重设计 §6 交互原则 3）。
- 命令面与 `--json`/退出码规范参照 git/kubectl 惯例；`--follow` 参照 `kubectl logs -f`。
- 首次运行脚本中 connect/import 的行为细节见组件设计 §2。
