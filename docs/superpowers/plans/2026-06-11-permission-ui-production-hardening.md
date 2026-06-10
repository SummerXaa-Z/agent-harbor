# Permission UI Production Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make AgentHarbor's Permission Changes workspace feel like a production B2B task flow, not a demo console, while preserving the core journey: configure -> approve -> apply -> status check -> acceptance.

**Architecture:** Keep the existing `AiAdminPermissionWorkbench` boundary, but introduce small presentation helpers inside that component before extracting files. The first pass fixes visible product semantics: one authoritative status, business-readable queue rows, a quieter first screen, and fewer technical identifiers in the primary path.

**Tech Stack:** React, TypeScript, Vite, existing CSS modules under `frontend/src/styles`, Node test runner, GitHub PR #74 branch `codex/production-readiness-gate`.

---

## Product Consensus

- Main users are platform admins and security reviewers, not protocol implementers.
- Primary path must show business terms: tenant name, workspace name, caller name, target name, permission package name, reviewer, and next action.
- Technical identifiers remain available only in advanced/details areas, copyable when necessary.
- The page must never imply two conflicting states at once. If an approval request is pending, the dominant status is pending approval, even if previous demo evidence exists.
- The first viewport should answer three questions: where am I, what is the current status, and what should I do next?
- Evidence, preflight internals, impact details, and audit material are secondary until the operator asks for them.

## Files

- Modify `frontend/src/components/AiAdminPermissionWorkbench.tsx`: status resolver, queue row rendering, first-screen hierarchy, copy cleanup.
- Modify `frontend/src/styles/permission-workbench.css`: compact task layout, queue row visual hierarchy, process nav polish, collapsible secondary content.
- Modify `frontend/src/i18n.ts`: Chinese and English copy for unified statuses, queue summaries, and advanced technical labels.
- Modify `frontend/tests/permissionFlowLayout.test.mjs`: structural regression tests for status precedence, technical-field hiding, queue row business labels, and first-screen focus.
- Modify `frontend/tests/i18n.test.mjs`: translation key parity and Chinese copy checks.
- Modify `docs/engineering/user-journey-review-2026-06-10.md`: record the sixth UI production hardening pass.
- Modify `CHANGELOG.md`: user-facing note under Unreleased.

## Task 1: Unified Journey Status

- [x] **Step 1: Write failing tests**

Add assertions to `frontend/tests/permissionFlowLayout.test.mjs`:

```js
test("permission request uses one authoritative journey status", () => {
  assert.match(workbench, /const journeyStatus = resolvePermissionJourneyStatus\(/);
  assert.match(workbench, /journeyStatus\.labelKey/);
  assert.match(workbench, /journeyStatus\.detailKey/);
  assert.match(workbench, /journeyStatus\.tone/);
  assert.doesNotMatch(workbench, /<Badge tone=\{draftStatus\.tone\}>\{t\(draftStatus\.labelKey\)\}<\/Badge>/);
  assert.match(workbench, /aria-label=\{t\("text.permissionJourneyStatus"\)\}/);
});
```

- [x] **Step 2: Run focused test and confirm failure**

Run:

```bash
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs
```

Expected: the new status resolver assertions fail before implementation.

- [x] **Step 3: Implement status resolver**

In `AiAdminPermissionWorkbench.tsx`, add a local helper near existing helper functions:

```ts
function resolvePermissionJourneyStatus(args: {
  approvalRequest: PermissionPackageApprovalRequest | null;
  canApply: boolean;
  draft: PermissionPackageDraft;
  goLiveReady: boolean;
  productionStatus: AiAdminProductionConsoleStatus;
  workbenchStatus?: PermissionPackageWorkbenchPreview["summary"]["status"];
}): { labelKey: string; detailKey: string; tone: Tone; nextActionKey: string } {
  if (args.goLiveReady) {
    return {
      detailKey: "permissionJourney.statusDetail.ready",
      labelKey: "permissionJourney.status.ready",
      nextActionKey: "action.exportProductionEvidence",
      tone: "success"
    };
  }
  if (args.approvalRequest?.status === "pending") {
    return {
      detailKey: "permissionJourney.statusDetail.awaitingApproval",
      labelKey: "permissionJourney.status.awaitingApproval",
      nextActionKey: "action.refreshReviewerQueue",
      tone: "warning"
    };
  }
  if (args.approvalRequest?.status === "rejected") {
    return {
      detailKey: "permissionJourney.statusDetail.rejected",
      labelKey: "permissionJourney.status.rejected",
      nextActionKey: "action.startPermissionApproval",
      tone: "danger"
    };
  }
  if (args.canApply) {
    return {
      detailKey: "permissionJourney.statusDetail.readyToApply",
      labelKey: "permissionJourney.status.readyToApply",
      nextActionKey: "action.applyPermissionPackage",
      tone: "accent"
    };
  }
  if (args.draft.readiness.missingFields.length > 0) {
    return {
      detailKey: "permissionJourney.statusDetail.needsInput",
      labelKey: "permissionJourney.status.needsInput",
      nextActionKey: "action.createApprovalRequest",
      tone: "warning"
    };
  }
  return {
    detailKey: "permissionJourney.statusDetail.needsApproval",
    labelKey: "permissionJourney.status.needsApproval",
    nextActionKey: "action.createApprovalRequest",
    tone: "warning"
  };
}
```

Use `journeyStatus` in the header and overview instead of mixing `workbenchStatusKey`, `draftStatus`, and `productionSummary.status`.

- [x] **Step 4: Add i18n**

Add English and Chinese keys:

```ts
"text.permissionJourneyStatus": "Permission journey status",
"permissionJourney.status.ready": "Ready for production",
"permissionJourney.status.awaitingApproval": "Waiting for approval",
"permissionJourney.status.rejected": "Approval rejected",
"permissionJourney.status.readyToApply": "Ready to apply",
"permissionJourney.status.needsInput": "Request needs input",
"permissionJourney.status.needsApproval": "Approval required",
"permissionJourney.statusDetail.ready": "Permission has been applied and production evidence is complete.",
"permissionJourney.statusDetail.awaitingApproval": "A reviewer must approve this request before permissions can be applied.",
"permissionJourney.statusDetail.rejected": "The reviewer rejected this request. Update the request before submitting again.",
"permissionJourney.statusDetail.readyToApply": "Approval and checks are ready. Apply the permissions to continue.",
"permissionJourney.statusDetail.needsInput": "Complete tenant, workspace, caller, target, and package selection first.",
"permissionJourney.statusDetail.needsApproval": "Submit the request for approval before applying permissions.",
```

Chinese copy should use:

```ts
"text.permissionJourneyStatus": "权限变更状态",
"permissionJourney.status.ready": "可上线",
"permissionJourney.status.awaitingApproval": "等待审批",
"permissionJourney.status.rejected": "审批已拒绝",
"permissionJourney.status.readyToApply": "可以应用",
"permissionJourney.status.needsInput": "申请待补充",
"permissionJourney.status.needsApproval": "需要审批",
"permissionJourney.statusDetail.ready": "权限已应用，上线证据已完成。",
"permissionJourney.statusDetail.awaitingApproval": "需要审批人通过后才能应用权限。",
"permissionJourney.statusDetail.rejected": "审批人已拒绝，请调整申请后重新提交。",
"permissionJourney.statusDetail.readyToApply": "审批和检查已就绪，可以应用权限继续推进。",
"permissionJourney.statusDetail.needsInput": "请先补齐租户、工作区、调用方、目标服务和权限包。",
"permissionJourney.statusDetail.needsApproval": "应用权限前需要先提交审批。",
```

- [x] **Step 5: Verify**

Run:

```bash
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs tests/i18n.test.mjs
```

Expected: all focused tests pass.

## Task 2: Hide Technical Fields From Primary Queue Rows

- [x] **Step 1: Write failing tests**

Add assertions:

```js
test("permission reviewer queue uses business labels before technical identifiers", () => {
  assert.match(workbench, /function permissionApprovalRequestBusinessLabel/);
  assert.match(workbench, /className="approval-review-row-main"/);
  assert.match(workbench, /className="approval-review-row-meta"/);
  assert.match(workbench, /<TechnicalId/);
  const queueStart = workbench.indexOf('<section className="approval-reviewer-queue"');
  const advancedStart = workbench.indexOf('<details className="approval-details"', queueStart);
  assert.notEqual(queueStart, -1);
  assert.notEqual(advancedStart, -1);
  assert.doesNotMatch(workbench.slice(queueStart, advancedStart), /request\.tenantId/);
  assert.doesNotMatch(workbench.slice(queueStart, advancedStart), /request\.callerInstanceId/);
});
```

- [x] **Step 2: Implement business labels**

In `AiAdminPermissionWorkbench.tsx`, add helper functions:

```ts
function permissionApprovalRequestBusinessLabel(
  request: PermissionPackageApprovalRequest,
  templates: PermissionPackageTemplate[],
  tenants: Tenant[],
  agents: Agent[],
  t: Translator
) {
  const template = templates.find((item) => item.id === request.packageId);
  const tenant = tenants.find((item) => item.id === request.tenantId);
  const caller = agents.find((item) => item.id === request.callerInstanceId);
  return {
    caller: caller ? permissionEntityDisplayName(caller.name, t) : t("text.unknownCaller"),
    template: template ? permissionPackageTemplateName(template, t) : t("text.unknownPermissionPackage"),
    tenant: tenant ? permissionEntityDisplayName(tenant.name, t) : t("text.unknownTenant")
  };
}
```

Render queue rows as:

```tsx
const queueLabel = permissionApprovalRequestBusinessLabel(request, templates, tenants, agents, t);
...
<span className="approval-review-row-main">
  <strong>{queueLabel.template}</strong>
  <small>{tx(t, "text.permissionQueueBusinessScope", { tenant: queueLabel.tenant, caller: queueLabel.caller })}</small>
</span>
<span className="approval-review-row-meta">
  {formatDate(request.expiresAt)}
</span>
```

Move raw IDs into `TechnicalId` inside advanced details.

- [x] **Step 3: Add i18n and styles**

Add keys for unknown labels and queue meta. Add CSS for `.approval-review-row-main`, `.approval-review-row-meta`, and row truncation using `min-width: 0`.

- [x] **Step 4: Verify**

Run:

```bash
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs tests/i18n.test.mjs
```

Expected: all focused tests pass and no queue row primary label contains raw IDs in browser smoke test.

## Task 3: First Viewport Task Flow

- [x] **Step 1: Write failing tests**

Assert the first screen has one primary status block and secondary technical/evidence areas are after the process panel:

```js
test("permission request first viewport prioritizes one task flow", () => {
  const headerStart = workbench.indexOf('<section className="approval-header"');
  const contextStart = workbench.indexOf('<section className="approval-context-bar"', headerStart);
  const overviewStart = workbench.indexOf('<section className="approval-overview"', contextStart);
  const flowStart = workbench.indexOf('<div className="approval-flow-layout">', overviewStart);
  assert.ok(headerStart >= 0 && contextStart > headerStart && overviewStart > contextStart && flowStart > overviewStart);
  assert.match(workbench, /className="approval-task-strip"/);
  assert.match(styles, /\.approval-task-strip\s*\{/);
  assert.match(styles, /\.approval-process-panel\s*\{[^}]*position:\s*sticky;/s);
});
```

- [x] **Step 2: Implement task strip**

Replace the metrics-heavy top area with a tighter task strip:

```tsx
<section className="approval-task-strip" aria-label={t("text.permissionRequestTaskStrip")}>
  <article>
    <span>{t("text.currentStatus")}</span>
    <strong>{t(journeyStatus.labelKey)}</strong>
    <small>{t(journeyStatus.detailKey)}</small>
  </article>
  <article>
    <span>{t("text.permissionRequestScope")}</span>
    <strong>{tenantPath.primary}</strong>
    <small>{workspaceName} · {callerName}</small>
  </article>
  <article>
    <span>{t("text.permissionRequestNextAction")}</span>
    <strong>{t(journeyStatus.nextActionKey)}</strong>
    <small>{permissionPackageTemplateName(draft.template, t)}</small>
  </article>
</section>
```

Keep detailed metrics below or inside secondary details, not as the dominant first-view content.

- [x] **Step 3: Verify layout**

Run focused tests and browser smoke:

```bash
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs
```

Browser expected: first viewport shows status, scope, next action, form, and sticky process panel without raw technical identifiers.

## Task 4: Copy, i18n, and Microinteraction Polish

- [x] **Step 1: Write failing tests**

Add checks that duplicated copy is removed:

```js
test("permission request copy avoids repeated step labels", () => {
  assert.doesNotMatch(i18n, /"permissionWorkbench\\.detail\\.approval_approved": "审批已通过且匹配当前申请。"/);
  assert.doesNotMatch(i18n, /"permissionWorkbench\\.detail\\.apply_applied": "权限已应用。"/);
  assert.match(i18n, /"permissionWorkbench\\.detail\\.approval_approved": "已通过，等待应用。"/);
  assert.match(i18n, /"permissionWorkbench\\.detail\\.apply_applied": "已应用，等待验证。"/);
});
```

- [x] **Step 2: Update copy**

Replace repeated Chinese text:

- `审批已通过且匹配当前申请。` -> `已通过，等待应用。`
- `权限已应用。` -> `已应用，等待验证。`
- `运行证据已完整。` -> `验证已通过。`
- `上线就绪检查已完成。` -> `证据已完成。`

Mirror concise English copy.

- [x] **Step 3: Add reduced-motion safe polish**

Only animate `box-shadow`, `border-color`, `background`, `color`, `opacity`, or `transform`; do not use `transition: all`.

- [x] **Step 4: Verify**

Run:

```bash
pnpm --dir frontend test
pnpm --dir frontend build
```

Expected: 0 failures and build succeeds.

## Task 5: Docs, Browser Verification, and PR Update

- [x] **Step 1: Update docs**

Append a sixth pass to `docs/engineering/user-journey-review-2026-06-10.md`:

- unified status model
- business-readable approval queue rows
- first viewport task strip
- technical IDs moved to advanced details
- focused and browser verification

Update `CHANGELOG.md` under Unreleased.

- [x] **Step 2: Run full gates**

Run:

```bash
git diff --check
make check
make release-check
```

Expected: all pass.

- [x] **Step 3: Browser smoke test**

Use the in-app browser at `http://127.0.0.1:5174/` and verify:

- first viewport has one status, one scope, and one next action
- process steps are clickable
- approval queue primary labels are business-readable
- no raw tenant/agent IDs appear in primary path
- Access Profile handoff still carries readable context

- [x] **Step 4: Commit and update PR**

Commit:

```bash
git add CHANGELOG.md docs/engineering/user-journey-review-2026-06-10.md frontend/src/components/AiAdminPermissionWorkbench.tsx frontend/src/i18n.ts frontend/src/styles/permission-workbench.css frontend/tests/i18n.test.mjs frontend/tests/permissionFlowLayout.test.mjs docs/superpowers/plans/2026-06-11-permission-ui-production-hardening.md
git commit -m "feat: harden permission journey ui"
git push
```

Expected: PR #74 updates with the new commit.

## Task 6: Responsive Production Shell Follow-Up

- [x] **Step 1: Verify tablet desktop width**

Use the in-app browser at 1024x768 and verify the Permission Changes primary journey:

- task strip remains full width
- process panel remains visible beside the form
- no horizontal overflow appears in header, context, task strip, form, or process panel

- [x] **Step 2: Fix responsive cascade**

Keep Permission Changes overrides after the global responsive rules in `frontend/src/styles.css`, so the old `max-width: 1120px` single-column rule does not collapse the production journey too early.

- [x] **Step 3: Verify mobile width**

Use the in-app browser at 390x844 and verify:

- context strip is single-column and readable
- task strip is single-column and readable
- process header only shows the authoritative journey status
- no horizontal overflow appears in the primary path

- [x] **Step 4: Update tests and docs**

Extend `frontend/tests/permissionFlowLayout.test.mjs` to lock the responsive cascade order, tablet two-column journey, mobile single-column strips, and removal of the approval-substatus badge from the journey header.

- [x] **Step 5: Align completed quick actions**

When `goLiveReady` is true, keep the primary action on evidence export and switch the header secondary action to Access Profile review. Browser verification confirmed the completed header reads `导出证据 / 查看权限画像` instead of sending users back to runtime validation.

## Task 7: Access Profile Handoff Tenant Readability

- [x] **Step 1: Browser-test the completed handoff**

Click the completed Permission Changes header action `查看权限画像` and verify the Access Profile workspace receives the tenant, workspace, caller, and target business context.

- [x] **Step 2: Move tenant ids out of the primary tenant-scope list**

Render tenant-scope rows with business tenant names and parent-tenant business context first. Keep `tenant.id` and `parentTenantId` inside an `高级设置` details section backed by the shared `TechnicalId` component.

- [x] **Step 3: Add regression coverage**

Extend `frontend/tests/permissionFlowLayout.test.mjs` to assert that Access Profile tenant rows import `TechnicalId`, build a tenant-name map, avoid raw ids in the primary row, and retain raw ids only in technical details.

- [x] **Step 4: Browser verification**

Browser verification confirmed tenant-scope rows render `客户服务中心` and `工单处理项目` in the primary row, with `rawInMain=false`, and technical ids available under `高级设置`.

- [x] **Step 5: Fold handoff technical filters**

Move Access Profile `tenantId`, `workspaceId`, `subjectId`, and trace limit inputs into `access-advanced-filters`. Browser verification confirmed the handoff query panel keeps target, capability, caller, load, and explain actions visible while advanced identifiers are collapsed by default.

## Task 8: Business Capability Names In Primary UI

- [x] **Step 1: Add shared capability presenter**

Use a single `capabilityDisplayName` presenter backed by localized capability-name keys, so demo and discovered capabilities can keep stable keys for API/audit while the console shows business-readable labels.

- [x] **Step 2: Replace primary-path capability keys**

Use the presenter in Permission Changes capability chips, Tenant Access Profile filter summary, grant rows, trace rows, Capability Governance pickers, capability catalog rows, grant-chain rows, approval-success messages, policy-gate reason messages, and Access Profile handoff context. Capability Governance grant-chain rows also map tenant ids to business tenant names in the primary line.

- [x] **Step 3: Keep technical keys out of the operator path**

Keep raw capability keys in the data model, tests, and technical/audit surfaces only. Do not rename API fields or persisted ids.

- [x] **Step 4: Add regression coverage**

Extend `frontend/tests/permissionFlowLayout.test.mjs` so future UI changes cannot reintroduce `capability.key` as the primary user-facing label.

- [x] **Step 5: Localize capability governance agent names**

Capability Governance now maps caller and target service names through the same business-name presenter, so table rows and selectors no longer show demo system names such as `Permission Package Approval MCP Target`.

- [x] **Step 6: Localize default context and capability summaries**

Permission Changes now treats the default tenant and workspace ids as business context in the primary UI, including empty API states, and localizes the default security approver display name in the approval step. Capability Governance now uses localized capability summaries, translated sensitivity/risk labels, and translated data-scope labels before falling back to technical descriptions.

- [x] **Step 7: Reuse business scope labels in access profile**

Tenant Access Profile now uses the shared data-scope label presenter, maps default tenant ids to business tenant names, and renders tenant hierarchy as user-readable levels instead of `L0` badges.

- [x] **Step 8: Localize go-live audit evidence**

Go-live Evidence now localizes management audit actions, resource types, actors, and summaries, and moves raw resource ids behind expandable technical details.

- [x] **Step 9: Promote go-live acceptance over historical evidence**

Go-live Evidence now starts with a go-live acceptance workflow that shows current permission-change context, readiness status, next actions, and production progress before historical evidence runs, and retitles the secondary table as evidence history.

- [x] **Step 10: Preserve workspace context in the URL**

The console now reflects the active workspace in the URL hash and restores that workspace on reload, so operators can refresh or reopen `#evidence` without losing the go-live acceptance context.

- [x] **Step 11: Align approval progress with applied evidence**

Go-live acceptance now treats applied or production-ready evidence as approval-satisfied, so the production progress list no longer shows "approval not requested" after permissions are active.

## Task 9: Completed Approval Step State

- [x] **Step 1: Treat applied changes as approval-resolved**

When a permission change is already applied or production-ready, render the main approval step as resolved even if an old pending approval queue item is still present in local state.

- [x] **Step 2: Remove stale pending actions from completed journeys**

Hide approve, reject, and withdraw actions from the main approval step once the permission change is applied or production-ready. The queue remains available in advanced evidence, but it no longer overrides the primary journey status.

- [x] **Step 3: Verify in browser**

Browser verification confirmed the completed Permission Changes approval block reads `已批准 / 当前权限变更已记录审批。`, with no `待审批` status and no approve/reject/withdraw actions in the main path.

- [x] **Step 4: Separate step title from action wording**

The Simplified Chinese approval step title now reads `审批处理` while the primary action remains `提交审批`, so a completed journey no longer looks like it is asking the operator to submit again.

- [x] **Step 5: Align runtime result with production-ready state**

When the journey is production-ready, the runtime result line now renders the completed validation copy instead of the pending `执行运行验证` copy, even if the local runtime result object was not retained.

- [x] **Step 6: Make applied state explicit**

Once an application record exists, the apply button is disabled and reads `已应用` instead of keeping the active `应用权限` wording.
