# 0.2.0 Production Journey Acceptance Design

## Context

AgentHarbor now has the main production primitives for v0.2.0:

- login control for authenticated management sessions;
- first-run setup routing;
- Resource Management as the Agent/MCP lifecycle entry;
- Access Query as the answer-first daily entry;
- Permission Changes for approval, apply, runtime validation, status check, and acceptance handoff;
- Go-Live Status for final review and handoff material.

The remaining risk is not a missing backend capability. The risk is that a real operator can still experience these as separate workspaces instead of one dependable production path. For v0.2.0, the product needs one verified journey:

```text
sign in
  -> prepare tenant and resources
  -> ask whether access is allowed
  -> fix denied access through a permission change
  -> approve and apply
  -> run validation
  -> confirm go-live status
  -> hand off the acceptance record
```

中文版目标：

```text
登录
  -> 准备租户与资源
  -> 查询访问是否允许
  -> 通过权限变更修复拒绝访问
  -> 审批并应用
  -> 运行验证
  -> 确认上线状态
  -> 交接验收记录
```

## Design Consensus

Build a production-journey acceptance layer. Do not add another product area, do not duplicate the Permission Changes workflow, and do not introduce a new backend service. The work should connect and verify the existing path.

Recommended direction:

1. Add a pure frontend journey model that derives the current production path stage from existing console data and permission-workbench state.
2. Surface that model as a quiet journey checkpoint in the relevant workspaces, so users know where they are and what the next safe action is.
3. Tighten cross-workspace handoff rules so Resource Management, Access Query, Permission Changes, and Go-Live Status preserve business context without exposing raw ids in the primary path.
4. Remove remaining user-facing "evidence" and "证据" wording from the console UI. Use "records", "acceptance", "status", "runtime log", or "handoff material" depending on context. Internal API names can stay unchanged for compatibility.
5. Add tests that lock the production journey path, user-facing wording, and no-new-dependency boundary.
6. Validate the final path with existing repository gates plus a browser smoke run.

## Product Behavior

### Empty System

When the system has no usable setup, the console should land on Getting Started. The user sees the next setup step and can move to Resource Management to register resources. This route must not pretend that permission changes can be made before tenants, callers, targets, capabilities, and grant context exist.

### Configured System

When setup is complete, the console should land on Access Query. The primary question is: can this caller access this target capability in this tenant context?

If the answer is denied, the user can start a permission fix. The handoff to Permission Changes is one-time, visible, and never auto-submits. Permission Changes remains the only place that approves, applies, validates, checks status, and prepares handoff material.

### Completed Permission Change

Once permissions are applied and validation/status checks pass, the user should see a clear completion state and three exits:

- confirm go-live status;
- review the tenant access profile;
- start a new permission change in the same business context.

The Go-Live Status workspace should present the current permission-change context first, then the readiness decision, then historical records. Empty historical lists must not dominate the page.

## UI Language

User-facing copy should avoid the criminal-investigation feel of "evidence" / "证据". Recommended replacements:

- "验收记录" for acceptance report/history;
- "运行记录" for runtime traces;
- "审计记录" for audit rows;
- "交接材料" for exportable handoff material;
- "上线状态" for the navigation and readiness page;
- "验收明细" for expandable operational details inside Permission Changes.

English should use "acceptance", "records", "runtime logs", "audit records", and "handoff material" instead of visible "evidence" labels. Internal code identifiers such as `productionEvidence` and existing REST/MCP tool names are not renamed in this slice.

## Technical Design

### Journey Model

Create a pure module, likely `frontend/src/productionJourney.ts`, that derives:

- current stage;
- completed stages;
- next action key;
- relevant target hash;
- whether the current system is empty, partially configured, configured, in-change, ready, or blocked.

Inputs should be existing structured data:

- setup readiness from `gettingStarted.ts`;
- resource lifecycle summary from `resourceLifecycle.ts`;
- access-query result/handoff state from `askJourney.ts`;
- permission-workbench status from existing permission presenters;
- readiness/application state already loaded by `ConsoleController`.

The model must not perform API calls, mutate state, or inspect DOM.

### UI Integration

Keep the visual layer restrained:

- Add a compact journey checkpoint treatment used by Getting Started, Resource Management, Access Query, Permission Changes, and Go-Live Status.
- Do not add another dashboard.
- Do not add nested cards.
- Make the current stage, next action, and carried business context visible.
- Keep technical ids in details/advanced controls.

The checkpoint should render as a thin path/status strip or existing header extension, not as a large standalone card. It should answer "where am I and what is next" without competing with the page's primary form or table.

### Handoff Rules

Handoffs should be explicit and one-time:

- Resource Management can send the user to Access Query or Permission Changes with business context.
- Access Query can prefill Permission Changes only after the user clicks the denied-access fix action.
- Permission Changes can send completed context to Go-Live Status and Tenant Access Profile.
- Go-Live Status can return to the same Permission Change without losing context.

No handoff should auto-submit, auto-approve, auto-apply, or widen permissions.

### Verification

Add focused tests for:

- empty system lands on Getting Started;
- configured system lands on Access Query;
- denied Access Query handoff goes to Permission Changes without auto-submit;
- completed Permission Changes exposes Go-Live Status and Tenant Access Profile exits;
- user-visible i18n labels do not expose "evidence" or "证据";
- no new frontend dependencies are introduced.

Use existing gates:

- `pnpm --dir frontend test`
- `pnpm --dir frontend build`
- `git diff --check`
- `make check`
- `make release-check`

Browser smoke should verify two paths:

1. Empty demo opens on Getting Started and guides to Resource Management.
2. Configured demo opens on Access Query, denied access can hand off to Permission Changes, the journey reaches ready, and Go-Live Status keeps the same business context.

## Boundaries

In scope:

- frontend journey model;
- compact journey checkpoint UI;
- i18n wording cleanup for visible labels in both English and Simplified Chinese;
- cross-workspace handoff tightening;
- docs and CHANGELOG;
- focused tests plus repository gates;
- browser smoke record.

Out of scope:

- backend API changes;
- database migrations;
- new authentication model;
- new browser automation dependency;
- renaming public REST or MCP tool identifiers;
- rollback execution;
- full resource CRUD redesign.

## Design Pressure Test

Key decisions after pressure-testing the design:

- **Should this add a new "Production Acceptance" page?** No. A new page would hide the problem instead of fixing the journey. Go-Live Status already owns the final gate; this slice should connect existing pages.
- **Should the checkpoint be visually prominent?** Only enough to orient the user. A large card would compete with forms and tables. Use a thin status/path treatment and existing page structure.
- **Should this rename backend `evidence` APIs?** No. Public REST/MCP names stay stable in v0.2.0. This slice only changes user-facing product language.
- **Should this introduce Playwright or another browser dependency?** No. The repository currently relies on Node tests and shell scenarios. Browser verification remains a recorded smoke run for this slice, while pure journey decisions are covered by unit tests.
- **Should the journey model decide permissions?** No. Permission decisions remain in backend/readiness evaluators and existing presenters. The journey model only derives operator stage, next action, and navigation context.

## Risks

- The console already has many journey-related helpers. The new model must compose existing pure functions instead of creating a second state machine.
- Removing visible "evidence" / "证据" labels must not break API compatibility or test fixtures that still use internal evidence terminology.
- A checkpoint component can become visual clutter. It should be compact and contextual, not a new dashboard header on every page.
- Browser screenshot tooling may be unavailable or time out. The acceptance record should distinguish DOM/interaction verification from screenshot evidence.

## Acceptance Criteria

- A non-technical administrator can understand the next safe action from each primary workspace without reading raw ids.
- The empty-system path and configured-system path have explicit, tested defaults.
- Access Query to Permission Changes to Go-Live Status keeps business context across page changes.
- User-facing labels no longer use visible "evidence" or "证据"; internal API/tool identifiers remain compatible.
- The implementation adds no backend dependency and no frontend package dependency.
- `make check` and `make release-check` pass before PR creation.
