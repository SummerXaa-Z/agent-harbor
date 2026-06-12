# Permission Changes Review Repair Report

- **Date / 日期**: 2026-06-10
- **Scope / 范围**: follow-up repair for `docs/engineering/permission-changes-review-2026-06-09.md`
- **Product standard / 产品标准**: keep the first production user journey usable, secure, stable, bilingual, and verifiable before any open-source release claim.
- **Primary journey / 主旅程**: AI Admin configures a permission change from a package template -> approval -> apply -> runtime verification -> production readiness -> bounded evidence handoff.

## Executive Summary / 总结

All blocker/high findings from the June 9 review are closed, and the remaining medium/low findings have been either fixed directly or converted into executable regression tests.

本轮已关闭 6 月 9 日审查中的全部 blocker/high 项；剩余 medium/low 项已直接修复，或沉淀为可执行回归测试。

The repair work deliberately avoided a cosmetic-only pass. It tightened write-path security, made fallback/sample states explicit and non-mutating, localized user-facing failures, replaced brittle source-grep coverage with executable tests where practical, and split the largest UI panels out of the app shell.

本轮不是只做界面美化，而是同步收紧写入路径安全、明确兜底数据且禁用写操作、本地化用户错误、把薄弱源码 grep 覆盖替换为可执行测试，并把最大 UI 面板从主壳中拆出。

## Finding Closure Matrix / Finding 收口表

| Finding | Severity | Status | Repair Summary / 修复摘要 | Evidence / 证据 |
|---|---:|---|---|---|
| d1-1 fallback sample data hidden | high | Closed | Permission Changes now shows persistent fallback warning and disables approval/apply/validation mutation actions when live data is unavailable. | `frontend/src/components/AiAdminPermissionWorkbench.tsx`, `frontend/tests/permissionFlowLayout.test.mjs` |
| d1-3 no initial loading state | medium | Closed | Added `aria-busy`, loading copy, and skeleton state before console data resolves. | `frontend/src/App.tsx`, `frontend/src/styles.css`, `frontend/tests/styleTheme.test.mjs` |
| d1-4 success/error messages share gray style | medium | Closed | Inline messages now map to semantic `success`, `warning`, `error`, and `info` classes. | `frontend/src/components/AiAdminPermissionWorkbench.tsx`, `frontend/src/styles/permission-workbench.css`, `frontend/tests/styleTheme.test.mjs` |
| d1-5 nav lacks `aria-current` | medium | Closed | Primary navigation now exposes `aria-current="page"`. | `frontend/src/App.tsx`, `frontend/tests/styleTheme.test.mjs` |
| d1-6 custom dropdown lacks labels/keyboard | medium | Closed | `ApprovalDropdown` now has labelled combobox semantics, active descendant wiring, and Arrow/Home/End/Escape/Enter/Space behavior. | `frontend/src/components/ApprovalDropdown.tsx`, `frontend/src/dropdownKeyboard.ts`, `frontend/tests/dropdownKeyboard.test.mjs`, `frontend/tests/permissionFlowLayout.test.mjs` |
| d2-1 empty or bare `*` subject selector | medium | Closed | Permission package drafts and instance assignments reject empty or bare wildcard selectors before writes. | `internal/permissionpack`, `internal/httpapi`, `frontend/tests/permissionPackages.test.mjs`, backend tests |
| d2-2 instance assignment empty selector | medium | Closed | Instance assignment creation enforces explicit bounded subject selectors. | `internal/httpapi/server.go`, store/app tests |
| d3-2 Chinese sessions leak English catch fallback | low | Closed | Error fallbacks now go through active-language translation keys; tests assert old hard-coded English strings stay out of `App.tsx`. | `frontend/src/App.tsx`, `frontend/src/i18n.ts`, `frontend/tests/i18n.test.mjs` |
| d3-3 raw data-scope enum labels | low | Closed | Data scopes can render business-readable localized labels including region and classification. | `frontend/src/accessProfile.ts`, `frontend/tests/accessProfile.test.mjs` |
| d4-2/d4-4 oversized app/styles | medium | Closed for requested slice | Split Tenant Access Profile and Capability Governance into dedicated components, moved shared presenter helpers and dropdown keyboard logic, and removed stale selectors. | `frontend/src/components/TenantAccessProfileView.tsx`, `frontend/src/components/CapabilityGovernanceView.tsx`, `frontend/src/consolePresenters.ts`, `frontend/tests/permissionFlowLayout.test.mjs`, `frontend/tests/styleTheme.test.mjs` |
| d5-6 mixed go-live terms | high | Closed | Product UI and docs now distinguish action `上线就绪检查`, status `上线就绪状态`, and evidence handoff `上线证据`. | `frontend/src/i18n.ts`, `README.md`, engineering docs |
| d6-1 self-reported reviewer and self-approval | high | Closed | Reviewers can be bound to authenticated admin identities; approve/reject rejects self-approval and enforces reviewer route scope. | `internal/httpapi/server.go`, `internal/app`, backend tests |
| d6-2 match-all subject selector | high | Closed | Empty and bare `*` selectors are rejected on draft/preflight/apply and grant creation paths. | `internal/permissionpack`, `internal/httpapi`, frontend/backend tests |
| d6-3 capability drift after approval | high | Closed | Approval snapshots include per-capability configuration fingerprints; preflight/apply rejects stale approvals after template, policy, or capability drift. | `internal/permissionpack`, `internal/httpapi/server_test.go` |
| d6-4 no admin key means management open | medium | Closed | Management routes fail closed unless an admin key, named identities, or explicit unauthenticated development flag is configured. | `internal/httpapi/server.go`, `make production-hardening`, release checks |
| d7-5 access object -> subject selector not executed | medium | Closed | Extracted reducer-like helper and tests for access-object selection; Capability Governance now also uses business access-object picker. | `frontend/src/permissionRequestForm.ts`, `frontend/src/components/CapabilityGovernanceView.tsx`, `frontend/tests/permissionRequestForm.test.mjs`, `frontend/tests/permissionFlowLayout.test.mjs` |
| d7-7 version drift apply boundary not tested | medium | Closed | Added backend test for template and policy version drift rejection on apply. | `internal/httpapi/server_test.go` |
| d7-2 five-step flow only source-grep tested | medium | Closed | Extracted executable process-step status helper and tests for current/completed states. | `frontend/src/permissionRequestJourney.ts`, `frontend/tests/permissionRequestJourney.test.mjs` |
| d7-1/3/4 weak i18n/style/API tests | low | Closed | Added translation key-set parity, localized fallback scan, style regression, dropdown behavior, and structure tests. | `frontend/tests/i18n.test.mjs`, `frontend/tests/styleTheme.test.mjs`, `frontend/tests/dropdownKeyboard.test.mjs`, `frontend/tests/permissionFlowLayout.test.mjs` |

## Security Repairs / 安全修复

### 1. Authenticated Reviewer Identity and Separation of Duties

Approval reviewer identity is no longer trusted as an arbitrary body string in production deployments. Named administrators can be configured through `AGENT_HARBOR_ADMIN_IDENTITIES`, and approval resolution rejects self-approval. Reviewer route rules still control tenant-subtree and workspace scope, but now they operate on an authenticated reviewer boundary instead of a self-reported label.

审批人身份不再在生产部署中依赖请求体自报。生产环境可以通过 `AGENT_HARBOR_ADMIN_IDENTITIES` 绑定具名管理员，审批处理会拒绝自审批。审批人路由仍用于限制租户子树和工作区范围，但现在建立在已认证身份之上。

### 2. Explicit Subject Binding

Permission package drafts, preflight/apply paths, and instance-assignment creation now reject empty subject selectors and bare `*`. This prevents an omitted field from becoming a match-all subject permission.

权限包草案、预检/应用路径和实例授权创建都会拒绝空主体选择器与裸 `*`，避免字段遗漏被解释为匹配全部主体。

### 3. Approval Drift Protection

Approval snapshots include capability configuration fingerprints in addition to draft id, template version, policy version, scope, allowed capability ids/keys, data scopes, request text, and subject selector. If a capability's effective permission configuration changes after approval, preflight/apply must re-request approval.

审批快照除草案 ID、模板版本、策略版本、作用域、允许能力 ID/key、数据范围、请求文本和主体选择器外，还包含能力配置指纹。审批后能力有效权限配置变更时，预检/应用必须重新审批。

### 4. Fail-Closed Management Defaults

Management routes no longer silently open when no admin key is set. Local development can still opt in with `AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true`, and release validation covers this boundary.

未配置 admin key 时，管理路由不再静默开放。本地开发仍可通过 `AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true` 显式开启，发布验证也覆盖该边界。

## UI and Product Repairs / UI 与产品修复

### 1. Production Task Flow

The Permission Changes workspace remains centered on one production task instead of many equal-weight evidence cards. The main flow keeps current status, next action, request configuration, approval/apply handling, runtime validation, and production readiness visible, while advanced evidence stays collapsed.

权限变更工作区继续围绕一个生产任务组织，而不是把证据卡片平铺为同级对象。主流程保留当前状态、下一步、变更配置、审批/应用处理、运行验证和上线就绪检查；高级证据默认折叠。

### 2. Business-Readable Access Objects

The primary Permission Changes form and Capability Governance direct-grant form now use role, department, and member access objects before translating them into technical subject selectors. Raw selector strings remain available only for advanced/custom cases.

权限变更主表单和能力治理直接授权表单都优先选择角色、部门和成员访问对象，再转换为技术主体选择器。原始 selector 字符串只保留给高级/自定义场景。

### 3. Accessible Interaction States

The app shell exposes `aria-busy` during initial loading. Navigation uses `aria-current="page"`. The five-step permission flow uses `aria-current="step"`. Custom dropdowns have accessible labels and keyboard movement.

产品外壳在初始加载时暴露 `aria-busy`；导航使用 `aria-current="page"`；五步权限流程使用 `aria-current="step"`；自定义下拉具备可访问标签和键盘移动。

### 4. Bilingual, Non-Leaky Errors

Catch-block fallbacks now map to i18n keys. Chinese sessions receive Chinese failure messages instead of generic English text. English remains available through the same key set, and tests assert English/Simplified Chinese key parity.

catch 兜底文案现在映射到 i18n key。中文会话收到中文失败提示，而不是通用英文字符串。英文仍通过同一 key 集合提供，测试会校验中英文 key 集合一致。

## Structural Repairs / 结构修复

The App shell was reduced from roughly 5,063 lines to roughly 4,294 lines in this repair slice. The split is intentionally conservative:

本轮将 App 壳从约 5,063 行压到约 4,294 行。拆分保持保守：

- `TenantAccessProfileView` owns the tenant access profile UI, filter form, grant-chain view, decision explanation, and trace evidence.
- `CapabilityGovernanceView` owns the capability catalog, grant-chain form, business access-object picker, and assignment summary.
- `consolePresenters.ts` owns shared formatting, status-tone, entity-name, and data-scope presenters.
- `dropdownKeyboard.ts` owns dropdown keyboard state transitions.
- `permissionRequestJourney.ts` owns executable five-step process state derivation.

This does not fully complete the long-term `useConsoleController()` extraction, but it does close the review-requested high-value split before further UI work lands.

这还没有完成长期的 `useConsoleController()` 提取，但已经完成审查报告要求的高收益拆分，避免后续 UI 继续堆进主文件。

## Test Coverage Added / 新增测试覆盖

Frontend test count increased from the earlier 71/73 range to 87 passing Node tests.

前端测试从此前约 71/73 条提升到 87 条全绿。

New coverage includes:

- Dropdown keyboard behavior and labelled combobox semantics.
- Access-object to subject-selector form derivation.
- Business-readable data-scope rendering.
- English/Simplified Chinese i18n key parity and localized error fallback scanning.
- Component split regression for Tenant Access Profile and Capability Governance.
- Capability Governance picker regression to avoid native select chrome.
- Five-step permission process status derivation.
- Demo-era CSS selector removal.
- Template/policy version drift apply rejection.

新增覆盖包括：

- 下拉键盘行为和带标签 combobox 语义。
- 访问对象到主体选择器的表单派生。
- 数据范围的业务可读展示。
- 中英文 i18n key 对齐和错误兜底本地化扫描。
- 租户访问画像与能力治理组件拆分回归。
- 能力治理选择器回归，避免回到原生 select。
- 五步权限处理状态推导。
- Demo 时代死样式选择器移除。
- 模板/策略版本漂移后的应用拒绝。

## Verification Plan / 验证计划

The final verification for this repair batch must include:

本轮最终验证必须包含：

```bash
pnpm --dir frontend test
pnpm --dir frontend exec tsc -b --pretty false
pnpm --dir frontend build
go test ./internal/httpapi -run 'TestPermissionPackageApprovalRejectsTemplateAndPolicyVersionDrift|TestPermissionPackageApprovalRejectsCapabilityDriftAfterApproval' -count=1
git diff --check
make gofmt-check
make check
make release-check
```

Browser smoke should verify:

浏览器烟测应验证：

- Permission Changes opens as the default workspace.
- No blank screen or console error after the component split.
- The primary permission-change flow remains visible and Chinese labels load.
- The fallback warning appears when live data is unavailable and mutation actions are disabled.
- Capability Governance uses in-app pickers for target, capability, caller, and access object.
- Access Profile loads through the split component.

## Verification Results / 实际验证结果

All required local gates passed after the repair batch.

本轮修复后，必要本地门禁全部通过。

```bash
pnpm --dir frontend test
# 87 tests passed

pnpm --dir frontend exec tsc -b --pretty false
# passed

pnpm --dir frontend build
# passed

go test ./internal/httpapi -run 'TestPermissionPackageApprovalRejectsTemplateAndPolicyVersionDrift|TestPermissionPackageApprovalRejectsCapabilityDriftAfterApproval' -count=1
# passed

git diff --check
# passed

make gofmt-check
# passed

make check
# passed

make release-check
# passed
```

The isolated browser journey also passed with non-default ports so it would not interrupt the user's existing local `5174` session:

```bash
AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true \
AGENT_HARBOR_BROWSER_GATE_API_PORT=9093 \
AGENT_HARBOR_BROWSER_GATE_FRONTEND_PORT=5176 \
MOCK_MCP_PORT=8788 \
make ai-admin-browser-journey
# passed
```

The first browser-gate attempt without `AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true` failed with `401 admin authentication is required`. That failure is expected under the new fail-closed management default, and the successful rerun used the explicit local-development flag.

第一次未设置 `AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true` 的浏览器门禁以 `401 admin authentication is required` 失败。这符合新的 fail-closed 管理鉴权默认值；成功重跑时显式开启了本地开发 flag。

The in-app Browser plugin itself could not attach to a webview during manual smoke (`Timed out waiting for the Browser webview to attach`). Because of that tool limitation, browser verification relied on the repository's `make ai-admin-browser-journey` release-candidate gate rather than an ad hoc in-app screenshot.

本次手工 in-app Browser 插件无法 attach 到 webview，因此没有把插件截图作为证据；浏览器验证以仓库自带的 `make ai-admin-browser-journey` 发布候选门禁为准。

## Residual Risks / 剩余风险

- The long-term controller extraction remains future work. `App.tsx` is materially smaller, but it still owns a large amount of state and handlers.
- Current frontend tests are still mostly Node-level source and pure-function tests rather than full React DOM tests. The browser smoke and scenario gates remain necessary.
- Real customer MCP/data integrations still require customer-specific credentials, tenant data, and deployment hardening outside this local repair batch.

剩余风险：

- 长期 controller 提取仍是后续工作。`App.tsx` 已明显变小，但仍持有大量状态和 handler。
- 当前前端测试仍以 Node 层源码/纯函数测试为主，不是完整 React DOM 测试，因此浏览器烟测和场景门禁仍然必要。
- 真实客户 MCP/数据集成仍依赖客户侧凭据、租户数据和部署加固，不属于本地修复批次能完全证明的范围。
