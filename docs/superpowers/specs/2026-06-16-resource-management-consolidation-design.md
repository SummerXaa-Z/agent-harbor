# Resource Management Consolidation Design

## 中文摘要

本轮目标是把 **资源管理 / Resource Management** 从“一个展示资源生命周期的摘要页”升级成真正的资源操作入口。用户不应该在 Agent 注册表、路由治理、策略审查等多个页面之间寻找创建 Agent、创建密钥、轮换凭据和创建策略的入口。创建类动作集中到资源管理；其他页面保留查看、审查、禁用、跳转和上下文说明。

这不是重新设计后端资源模型，也不新增资源 API。现有 Agent、凭据、策略和能力治理 API 保持不变，本轮只做前端信息架构、弹窗表单、视觉和文案收束。

## Goal

Make the Resource Management workspace answer one operator question:

> Where do I register, connect, secure, authorize, and validate Agent/MCP resources?

The answer should be one visible workbench, not several scattered forms.

## Existing State

The repository already has:

- `frontend/src/resourceLifecycle.ts`: pure presenter for resource lifecycle status.
- `frontend/src/components/ResourceLifecycleView.tsx`: lifecycle metrics, stages, actions, and resource rows.
- `frontend/src/components/ConsolePrimitives.tsx`: `ActionModalButton` and modal shell.
- `frontend/src/components/ManagementForms.tsx`: Agent, key, credential, policy, and trace forms.
- `frontend/src/ConsoleController.tsx`: existing handlers from `useManagementOperations`.
- `#registry`, `#routes`, `#policies`, `#capabilities`, and `#traces` detail workspaces.

The gap is product coherence:

- `#registry` already exposes lifecycle action slots, but the buttons still look like passive small cards.
- `#routes` and `#policies` still expose create-policy actions, so creation remains distributed.
- Empty policy states try to open the local policy form instead of routing the user to the resource lifecycle entry.
- Modal forms use a generic panel body. They work, but do not feel like focused B2B task dialogs.
- User-facing copy still contains some “evidence/证据” terms in visible product language.

## Design Consensus

Use the existing Resource Management page as the single command center for resource lifecycle writes.

1. **Centralize mutation entry points**
   - Keep create Agent, create key, rotate credential, and create policy in `#registry` only.
   - Remove create-policy triggers from `#routes` and `#policies`.
   - Policy empty state should route to `#registry`, not open a hidden local form.

2. **Make command actions look like actions**
   - Resource command buttons should use primary/secondary button affordances, not broad passive cards.
   - The first action, create Agent, is the primary action.
   - Supporting actions are secondary but still visually clickable.

3. **Upgrade modal forms**
   - Keep the existing modal mechanism and handlers.
   - Add a form-oriented modal variant with clear max width, structured body, sticky footer behavior where useful, and compact input spacing.
   - Keep keyboard escape, backdrop close, focus return, and focus-visible behavior.

4. **Keep detail pages focused**
   - `#routes` and `#policies` are review/governance pages.
   - They can disable or review existing rows, but should not be the primary creation surface.
   - They may link back to `#registry` for lifecycle changes.

5. **Remove detective-style wording from visible product copy**
   - Avoid “evidence/证据” in primary UI labels.
   - Use “records”, “acceptance report”, “go-live check”, “validation”, “handoff materials”, “运行记录”, “验收报告”, “上线检查”, “交接材料”.
   - Keep internal type names such as `EvidenceRun` and API method names when changing them would be large or risky.

## User Experience

`#registry` should open with:

1. A concise lifecycle overview.
2. A command bar:
   - Primary: Create Agent.
   - Secondary: Create key, rotate credential, create policy.
   - Navigation: review capabilities, start permission change, review runtime.
3. Resource lifecycle table.
4. Agent inventory and contract matrix as supporting detail.

`#routes` should show route governance and runtime records. If there are no policies, the empty state should explain that policy creation starts from Resource Management and offer a link.

`#policies` should show policy review, capability governance context, and management audit. If there are no policies, it should also route the user to Resource Management instead of opening a local policy form.

## Technical Design

Frontend-only changes:

- `ConsolePrimitives.tsx`
  - Extend `ActionModalButton` with a `tone?: "primary" | "secondary"` prop for the command variant.
  - Keep `variant="compact"` unchanged for existing small panel actions.
  - Keep `variant="command"` for resource lifecycle actions, but style it as a true button.

- `ConsoleController.tsx`
  - Make `createAgentAction("command", "primary")` the first command.
  - Render create key, rotate credential, and create policy as secondary command buttons.
  - Stop passing `createPolicyAction()` into `RoutesView` and `PoliciesView`.
  - Ensure policy empty-state CTA links to `#registry`.

- `OperationalViews.tsx`
  - Replace local DOM form-opening behavior in `AccessPolicyWorkspace` with a normal link/button to `#registry`.
  - Keep route governance and audit disclosure behavior.

- `styles.css`
  - Redesign `.resource-lifecycle-command-center`, `.action-modal-trigger-command`, and `.action-modal-panel`.
  - Improve form rhythm inside modal bodies without changing every form component.
  - Preserve existing tokens and avoid raw component colors.

- `i18n.ts`
  - Add or adjust EN + zh-CN copy for resource action guidance and policy empty states.
  - Replace visible “evidence/证据” labels with production-language alternatives.

## Security And Stability

- No backend changes.
- No permission semantics changes.
- No new dependencies.
- No automatic form submission from navigation or empty states.
- Modals continue to use existing write handlers and live-API guards.
- User-entered form state remains in existing management hook state; this avoids accidental reset during modal styling.

## Testing

Add source-level and behavior guard tests:

- Resource Management remains the single lifecycle mutation entry.
- `#routes` and `#policies` do not render create-policy modal triggers.
- Policy empty state routes to `#registry`.
- Command modal triggers support primary/secondary tones.
- Command styling remains token-based and clickable.
- Visible Chinese product copy does not include “证据”.
- English primary UI labels do not use “evidence” for main user-facing labels.

Run:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/consoleNavigation.test.mjs
pnpm --dir frontend test
pnpm --dir frontend build
make check
make release-check
```

## Non-Goals

- Do not remove the `#routes`, `#policies`, `#capabilities`, or `#traces` routes.
- Do not rename backend API fields or persisted model names containing `evidence`.
- Do not redesign the full permission-change workbench.
- Do not add a resource CRUD backend.
- Do not introduce a new design system or dependency.
