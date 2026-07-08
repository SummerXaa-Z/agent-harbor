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

When `goLiveReady` is true, keep the primary action on report export and switch the header secondary action to Access Profile review. Browser verification confirmed the completed header reads `导出验收报告 / 查看权限画像` instead of sending users back to runtime validation.

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

## Task 10: Browser Gate Matches Production Admin Identity Boundary

- [x] **Step 1: Reproduce the gate mismatch**

`make ai-admin-browser-journey` was rerun while the default local API on `9090` was active. The first run failed because the default port was occupied; the isolated-port rerun then failed with `401 admin authentication is required`; the next rerun with a shared admin key failed with `403 reviewer must match authenticated admin identity`. These failures showed that the release-candidate browser gate no longer matched the production fail-closed admin identity model.

- [x] **Step 2: Make the gate use split identities by default**

`scripts/scenario-ai-admin-browser-journey.sh` now starts the API with default split admin identities:

```bash
requester=browser-gate-requester-key;security-reviewer=browser-gate-reviewer-key
```

The script passes the requester key to setup and apply calls, and passes the reviewer key to the approval action.

- [x] **Step 3: Verify reviewer impersonation is rejected**

`scripts/scenario-permission-package-approval.sh` now supports separate `REQUESTER_ADMIN_KEY` and `REVIEWER_ADMIN_KEY` values. When the keys differ, the scenario first tries to approve with the requester key while claiming the reviewer actor and expects `403 authenticated admin identity`; only after that does the real reviewer approve.

- [x] **Step 4: Document and verify**

Documentation was updated in `README.md`, `CHANGELOG.md`, `docs/engineering/release-checklist.md`, `docs/product/0.2.0-ai-admin-permission-journey.md`, and `docs/engineering/0.2.0-local-validation-record.md`.

Verified:

```bash
bash -n scripts/scenario-permission-package-approval.sh scripts/scenario-ai-admin-browser-journey.sh
AGENT_HARBOR_BROWSER_GATE_API_PORT=19090 AGENT_HARBOR_BROWSER_GATE_FRONTEND_PORT=15174 MOCK_MCP_PORT=18787 make ai-admin-browser-journey
git diff --check
make scenario-scripts-lint
make makefile-targets-test
make check
make release-check
```

The browser journey passed with run id `ai-admin-browser-journey-20260611041605` and explicitly logged `approval reviewer impersonation rejected`.

## Task 11: Continue Main Journey UI Context Scan

- [x] **Step 1: Inspect the live Permission Changes route**

Use the in-app browser on `http://127.0.0.1:5174/#ai-admin` and verify the first viewport answers these in order: current workspace, selected tenant/workspace/caller/target/access object, current status, next action, and where acceptance evidence lives.

- [x] **Step 2: Inspect the Go-Live Acceptance route**

Use the in-app browser on `http://127.0.0.1:5174/#evidence` and verify the acceptance route keeps the same permission-change context, avoids duplicate status messages, and does not expose raw identifiers before business labels.

- [x] **Step 3: Choose the next smallest production fix**

Pick exactly one problem from the scan that breaks user confidence in the configure -> approve -> apply -> status check -> acceptance journey. The fix must include code, i18n if copy changes, docs, focused tests, browser verification, and release gates before commit.

The selected issue was Go-Live Acceptance's management audit table still feeling like a technical log: each row showed an "Advanced settings" disclosure in the primary scan path, and global badge styling forced mixed-case business words like `Agent` to lowercase. The fix changed audit technical-id disclosure text to `Details` / `详情`, preserved badge text casing, and verified in browser that raw ids remain hidden while audit rows read `创建 Agent ... 详情`.

Verified:

```bash
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs tests/i18n.test.mjs
```

Browser verification on `http://127.0.0.1:5174/#evidence` confirmed no visible raw ids, no `高级设置` audit-row text, and no lowercase `agent` audit action.

## Task 12: Continue Acceptance And Main Journey Scan

- [x] **Step 1: Re-check the Permission Changes route after audit polish**

Reload `http://127.0.0.1:5174/#ai-admin` and verify the first viewport still has one obvious next action, no duplicate status sources, no raw identifiers in the primary path, and no native select controls in the main journey.

- [x] **Step 2: Re-check the Acceptance route after audit polish**

Reload `http://127.0.0.1:5174/#evidence` and verify the management audit table stays secondary, the acceptance status and current permission-change context remain dominant, and technical detail disclosures are visually quiet.

- [x] **Step 3: Pick the next production-confidence fix**

Choose the smallest remaining issue that affects safety, stability, or user confidence; implement it with tests, docs, browser verification, `make check`, `make release-check`, commit, and push.

The selected issue was the completed Permission Changes journey still showing stale reviewer-queue rows with active approve/reject actions inside Acceptance Details. After permissions are already active or the status check is ready, those actions can make an operator think they still need to approve an old request. The queue now switches to read-only evidence: it keeps the business row and technical request ids, shows `Read-only` / `只读`, and removes approve/reject actions.

Verified:

```bash
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs tests/i18n.test.mjs
```

Browser verification on `http://127.0.0.1:5174/#ai-admin` confirmed the expanded Acceptance Details reviewer queue shows `权限已经生效，待审批队列仅用于追溯。`, has no `批准请求` or `拒绝请求` buttons, and marks pending rows as `只读`.

## Task 13: Continue Completion-State Cleanup

- [x] **Step 1: Inspect completed-state primary controls**

Reload `http://127.0.0.1:5174/#ai-admin` and verify completed-state controls do not mix active action styling with disabled completed labels, especially the `已应用` primary button and duplicate `导出证据` actions.

- [x] **Step 2: Inspect acceptance page after completed-state queue cleanup**

Reload `http://127.0.0.1:5174/#evidence` and verify the status check, current permission-change context, historical evidence, management audit, and runtime signals appear in a sensible order without inviting unnecessary writes.

- [x] **Step 3: Pick the next smallest production-confidence fix**

Prioritize safety and clarity over visual polish. If no critical main-journey issue remains, switch to CI status review and release notes cleanup.

The selected issue was the completed Permission Changes journey still rendering `已应用` as a disabled primary button. That looked like a blocked action rather than a completed state, and it competed visually with the actual next safe action, `导出证据`. Completed journeys now render `已应用` as a neutral success status indicator; the real primary actions remain active buttons.

Verified:

```bash
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs tests/i18n.test.mjs
```

Browser verification on `http://127.0.0.1:5174/#ai-admin` confirmed there is no visible `已应用` button, `.approval-action-status.is-complete` renders `已应用`, and the remaining primary actions are `导出证据`. Browser verification on `http://127.0.0.1:5174/#evidence` confirmed the acceptance route still keeps the current permission-change context, status check, and management audit in a sensible order with no visible raw UI ids.

## Task 14: Release-Gate And PR Status Sweep

- [x] **Step 1: Run whitespace and focused UI gates**

Run `git diff --check` and the focused Permission Changes/i18n test suite.

- [x] **Step 2: Run repository safety gates**

Run `make check` and `make release-check`.

- [x] **Step 3: Commit, push, and inspect PR checks**

Commit the completed-state status fix, push branch `codex/production-readiness-gate`, and inspect PR #74 check status. If checks are still queued, record that instead of claiming green CI.

Verified locally before commit:

```bash
git diff --check
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs tests/i18n.test.mjs
make check
make release-check
```

Committed and pushed `f888358 fix: render applied state as status` to `codex/production-readiness-gate`. GitHub PR #74 reported `mergeStateStatus=CLEAN`; `gh pr checks 74` showed Backend, Frontend, and PostgreSQL integration queued as pending immediately after push.

## Task 15: Continue Main Journey Production Scan

- [x] **Step 1: Wait for remote checks to settle**

Re-check PR #74 after the queued GitHub checks have had time to run. If any job fails, debug that failure before making more product changes.

PR #74 at `2c4c8dd` reported Backend and Frontend passing. PostgreSQL integration failed before running project tests because GitHub Actions could not pull `postgres:16` from Docker Hub (`context deadline exceeded` / `Client.Timeout exceeded while awaiting headers`). This was classified as an external runner/network failure, not a code failure.

- [x] **Step 2: Re-scan the first viewport and action set**

Use the in-app browser on `http://127.0.0.1:5174/#ai-admin` to verify the first viewport still has one clear next action, no duplicate primary controls, and no exposed technical identifiers.

- [x] **Step 3: Choose the next production-confidence increment**

If CI stays green, pick the next smallest issue that affects the configure -> approve -> apply -> status check -> acceptance path. Prefer removing ambiguity and accidental writes over adding new features.

The selected issue was the global connection settings menu staying open across SPA workspace navigation. When left open, technical fields such as `X-Admin-Key`, `tenantId`, and `workspaceId` remained visible at the top of business workspaces, which undermined the non-technical Permission Changes journey. The menu is now controlled by React state and closes whenever `activeNav` changes; the summary uses explicit state toggling so it remains keyboard/click operable.

Verified:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs
```

Browser verification opened `连接设置` on `http://127.0.0.1:5174/#ai-admin`, switched to `http://127.0.0.1:5174/#evidence`, and confirmed the connection menu was closed with no visible `X-Admin-Key`, `tenantId`, or `workspaceId` fields in the acceptance route.

## Task 16: Release-Gate Product Shell Cleanup

- [x] **Step 1: Run focused and repository gates**

Run `git diff --check`, `pnpm --dir frontend exec node --test tests/styleTheme.test.mjs`, `make check`, and `make release-check`.

- [ ] **Step 2: Commit, push, and inspect PR checks**

Commit the product shell connection-menu cleanup, push branch `codex/production-readiness-gate`, and inspect PR #74 checks.

- [ ] **Step 3: Continue only if checks are green or queued cleanly**

If CI fails, debug it before taking another product increment. If it is queued, record the state and continue with the next local product scan.

Verified locally before commit:

```bash
git diff --check
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs
make check
make release-check
```

Committed and pushed `dc5e063 fix: close connection settings on navigation` to `codex/production-readiness-gate`. GitHub PR #74 queued Backend, Frontend, and PostgreSQL integration for the new head.

## Task 17: Remove Duplicate Primary Completion Action

- [x] **Step 1: Re-scan the completed Permission Changes first viewport**

Browser scan of `http://127.0.0.1:5174/#ai-admin` showed two visible primary `导出证据` buttons: one in the task header and one in the completion card. The page had no visible raw ids and the connection menu was closed, but the duplicate primary CTA weakened the one-next-action model.

- [x] **Step 2: Convert completion-card export into a secondary exit**

The task header keeps the single primary next action, `导出证据`. The completion card now uses a secondary button labeled `下载验收报告`, alongside `查看权限画像` and `新建权限变更`.

- [x] **Step 3: Verify in focused tests and browser**

Verified:

```bash
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs tests/i18n.test.mjs
```

Browser verification confirmed `primaryButtons=["导出证据"]`, `exportPrimaryCount=1`, completion-card buttons are all `secondary-button`, and the completion report exit reads `下载验收报告`.

## Task 18: Release-Gate Duplicate Primary Cleanup

- [x] **Step 1: Run focused and repository gates**

Run `git diff --check`, `pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs tests/i18n.test.mjs`, `make check`, and `make release-check`.

- [ ] **Step 2: Commit, push, and inspect PR checks**

Commit the duplicate-primary cleanup, push branch `codex/production-readiness-gate`, and inspect PR #74 checks.

Verified locally before commit:

```bash
git diff --check
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs tests/i18n.test.mjs
make check
make release-check
```

Committed and pushed `165fa47 fix: keep one primary completion action` to `codex/production-readiness-gate`. GitHub PR #74 reported `mergeStateStatus=CLEAN`; Backend, Frontend, and PostgreSQL integration were pending immediately after push.

## Task 19: Remove Misleading Request Step Count

- [x] **Step 1: Inspect the completed process navigation**

Browser scan of `http://127.0.0.1:5174/#ai-admin` showed the request step as complete while still rendering `2/3`. That number came from backend workbench preview capability counts, not request form completion, and could make operators think the request was only partially complete.

- [x] **Step 2: Add backend regression coverage**

`TestPermissionPackageWorkbenchPreviewSummarizesPrimaryJourney` now asserts the `request` workbench step does not carry `count` / `total` capability ratios.

- [x] **Step 3: Remove the count from workbench preview**

`permissionPackageWorkbenchSteps` now returns the request step with status and detail code only. Capability counts remain available in the workbench summary and template/package areas where they are semantically meaningful.

- [x] **Step 4: Run release gates and commit**

Run focused backend and frontend tests, `make check`, `make release-check`, then commit and push if clean.

Verified locally before commit:

```bash
git diff --check
go test ./internal/httpapi -run TestPermissionPackageWorkbenchPreviewSummarizesPrimaryJourney -count=1
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs tests/i18n.test.mjs
make check
make release-check
```

Committed and pushed `f2ed00e fix: remove request step count noise` to `codex/production-readiness-gate`. GitHub PR #74 reported `mergeStateStatus=CLEAN`; Backend, Frontend, and PostgreSQL integration were pending immediately after push.

## Task 20: Hide Technical IDs In Access Profile Evidence

- [x] **Step 1: Validate the access-profile handoff**

Browser verification from `http://127.0.0.1:5174/#ai-admin` through `查看权限画像` confirmed the handoff context is preserved: tenant `客户服务中心`, workspace `客户服务工作区`, caller `客服助手`, and target `工单工具服务` all carry into the access profile.

- [x] **Step 2: Identify remaining technical-id noise**

The access-profile grant chain still showed raw grant ids such as `ent_...` and workspace ids such as `ws-permission-package-approval` in the main evidence rows, which made a production acceptance artifact look like a technical debug table.

- [x] **Step 3: Move technical ids into collapsed details**

Grant rows now show capability, target, effect, status, and data scope in the primary row. Tenant entitlement, target id, capability id, workspace assignment id, workspace id, and parent entitlement id are available through collapsed `TechnicalId` details.

- [x] **Step 4: Verify in focused tests and browser**

Verified:

```bash
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs tests/i18n.test.mjs
```

Browser verification confirmed the access profile reached through the Permission Changes journey has `hasVisibleRawGrantId=false`, `hasVisibleWorkspaceId=false`, and collapsed technical detail rows for the hidden ids.

## Task 21: Release-Gate Access Profile Evidence Cleanup

- [x] **Step 1: Run focused and repository gates**

Run `git diff --check`, `pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs tests/i18n.test.mjs`, `make check`, and `make release-check`.

Verified locally before commit:

```bash
git diff --check
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs tests/i18n.test.mjs
make check
make release-check
```

- [x] **Step 2: Commit, push, and inspect PR checks**

Commit the access-profile evidence cleanup, push branch `codex/production-readiness-gate`, and inspect PR #74 checks.

Committed and pushed `f3cc9a2 fix: hide access profile technical ids` to `codex/production-readiness-gate`. GitHub PR #74 reported `mergeStateStatus=CLEAN`; no checks were reported immediately after push.

## Task 22: Hide Instance Selectors In Access Profile Evidence

- [x] **Step 1: Inspect the remaining visible evidence noise**

Browser verification on `http://127.0.0.1:5174/#access` showed the grant and workspace technical ids were collapsed, but instance rows and trace rows still showed English demo names such as `Permission Package Approval Caller`; instance rows also showed subject selectors such as `user:support-*`, and trace rows showed raw backend reasons such as `capability assignment matched` in the primary evidence path.

- [x] **Step 2: Move instance technical fields into details**

Instance rows now render the business caller label and access-object label in the primary row. Trace rows render business caller, target, and localized decision-reason labels. Instance-assignment id, caller instance id, and subject selector remain available through collapsed `TechnicalId` details.

- [x] **Step 3: Verify focused tests and browser**

Run focused frontend tests and verify the access profile page no longer exposes the raw instance selector in the visible primary instance row.

Verified:

```bash
git diff --check
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs tests/i18n.test.mjs
```

Browser verification from Permission Changes through `查看权限画像` confirmed context `客户服务中心 / 客户服务工作区 / 客服助手 / 工单工具服务`; instance rows have `primaryHasSelector=false`; trace rows have `traceHasEnglishDemoName=false`, `traceHasRawReason=false`, and `traceHasLocalizedReason=true`.

- [x] **Step 4: Run repository gates**

Run `make check` and `make release-check`.

Verified locally before commit:

```bash
make check
make release-check
```

- [x] **Step 5: Commit, push, and inspect PR checks**

Commit and push if clean, then inspect PR #74.

Committed and pushed `67fb16a fix: localize access profile evidence rows` to `codex/production-readiness-gate`. GitHub PR #74 reported `mergeStateStatus=UNSTABLE` because Backend, Frontend, and PostgreSQL integration checks were still queued immediately after push.

## Task 23: Suppress Request Step Ratios In The UI

- [x] **Step 1: Inspect the visible regression**

Browser verification on `http://127.0.0.1:5174/#ai-admin` still showed `2/3` on the completed `填写申请` process step when the running API returned old workbench preview counts.

- [x] **Step 2: Defend in the frontend process mapping**

The workbench now discards `count` and `total` for the `request` step before rendering process navigation. Backend preview already omits these values, but the frontend also guards stale preview data.

- [x] **Step 3: Verify focused tests and browser**

Run focused frontend tests and verify the browser no longer shows `2/3` on `填写申请`.

Verified:

```bash
git diff --check
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs tests/i18n.test.mjs
```

Browser verification showed request step text `1填写申请已选择租户、调用方、工具服务和权限模板。` with `hasRatio=false`.

- [x] **Step 4: Run repository gates**

Run `make check` and `make release-check`.

Verified locally before commit:

```bash
make check
make release-check
```

- [x] **Step 5: Commit, push, and inspect PR checks**

Commit and push if clean, then inspect PR #74.

Committed and pushed `24a9cde fix: hide request step ratios` to `codex/production-readiness-gate`. GitHub PR #74 reported `mergeStateStatus=UNSTABLE` because Backend, Frontend, and PostgreSQL integration checks were in progress immediately after push.

## Task 24: Hide Closed Connection Settings Popover

- [x] **Step 1: Inspect the evidence page overlay**

Browser verification on `http://127.0.0.1:5174/#evidence` showed the connection settings popover over the business page even though `.connection-menu.open=false`. The popover exposed `X-Admin-Key`, tenant id, and workspace id on the go-live acceptance page.

- [x] **Step 2: Add explicit closed-state CSS**

Added `.connection-menu:not([open]) .connection-popover { display: none; }` so closed connection settings cannot render sensitive setup fields even if browser `details` default hiding is overridden by app styles.

- [x] **Step 3: Verify focused tests and browser**

Run focused shell tests and verify the evidence page keeps connection settings collapsed.

Verified:

```bash
git diff --check
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/permissionFlowLayout.test.mjs tests/i18n.test.mjs
```

Browser verification on `http://127.0.0.1:5174/#evidence` showed `.connection-menu.open=false`, `.connection-popover.display=none`, and `visibleAdminKeyLabel=false`.

- [x] **Step 4: Run repository gates**

Run `make check` and `make release-check`.

Verified locally before commit:

```bash
make check
make release-check
```

- [x] **Step 5: Commit, push, and inspect PR checks**

Commit and push if clean, then inspect PR #74.

Committed and pushed `3c858dd fix: hide closed connection popover` to `codex/production-readiness-gate`. GitHub PR #74 reported `mergeStateStatus=UNSTABLE` because Backend, Frontend, and PostgreSQL integration checks were queued immediately after push.

## Task 25: Promote Export Evidence On Ready Acceptance

- [x] **Step 1: Inspect the go-live acceptance primary action**

Browser verification on `http://127.0.0.1:5174/#evidence` showed the page was `可上线` and all checks were complete, but the first primary action remained `执行状态检查`; this made a completed acceptance page look like it still needed validation.

- [x] **Step 2: Switch primary and secondary actions by readiness**

`GoLiveAcceptanceOverview` now treats `productionReadiness?.status === "ready"` as the acceptance-ready state. When ready, `导出证据` is the primary action and `执行状态检查` is a secondary re-check action. When not ready, `执行状态检查` remains primary and export stays secondary.

- [x] **Step 3: Verify focused tests and browser**

Run focused frontend tests and verify the evidence page shows `导出证据` as the primary action in the ready state.

Verified:

```bash
git diff --check
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs tests/i18n.test.mjs
```

Browser verification on `http://127.0.0.1:5174/#evidence` showed status `可上线`, primary action `导出证据`, and secondary actions `执行状态检查` / `查看权限变更`.

- [x] **Step 4: Run repository gates**

Run `make check` and `make release-check`.

Verified locally before commit:

```bash
make check
make release-check
```

- [x] **Step 5: Commit, push, and inspect PR checks**

Commit and push if clean, then inspect PR #74.

Committed and pushed `9aba35a fix: prioritize report export when ready` to `codex/production-readiness-gate`. GitHub PR #74 reported `mergeStateStatus=UNSTABLE` because Backend, Frontend, and PostgreSQL integration checks were in progress immediately after push.

## Task 26: Make Runtime Audit Business Readable

- [x] **Step 1: Inspect the runtime audit workspace**

Browser verification on `http://127.0.0.1:5174/#traces` showed the trace page still behaved like an engineering console: the filter row exposed `runId`, caller/target options used English demo names, and trace rows surfaced `mcp:tools/call`, `cap_...`, and raw English decision reasons.

- [x] **Step 2: Move protocol details into advanced trace details**

The runtime audit filter now keeps run id in advanced settings, caller and target options use localized business names, and trace rows show business route labels such as tool discovery/tool invocation. Protocol route fields, capability id, and run id are available inside each row's technical details disclosure.

- [x] **Step 3: Share TechnicalId and localize trace reasons**

`TechnicalId` is now the shared copyable component used by dense workspaces. Runtime audit maps known backend trace reasons (`filtered tools/list by capability assignments`, `capability is not approved`) to localized operator-facing labels.

- [x] **Step 4: Verify focused tests and browser**

Verified:

```bash
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs tests/i18n.test.mjs tests/styleTheme.test.mjs
```

Browser verification on `http://127.0.0.1:5174/#traces` showed primary rows such as `客服助手 → 工单工具服务 / 允许 / 工具发现 / 工具列表已按权限收敛` and no visible `runId`, `mcp:tools/call`, `cap_`, or English demo names in the primary trace rows.

- [x] **Step 5: Run repository gates, commit, push, and inspect PR checks**

Run `git diff --check`, `make check`, and `make release-check`; then commit and push if clean.

Verified locally before commit:

```bash
git diff --check
make check
make release-check
```

Committed and pushed `7b1a678 fix: quiet runtime audit technical details` to `codex/production-readiness-gate`. GitHub PR #74 reported fresh Backend, Frontend, and PostgreSQL integration checks as pending immediately after push.

## Task 28: Separate Runtime Group And Workspace Copy

- [x] **Step 1: Inspect runtime navigation wording**

The sidebar used `运行审计` both as the navigation group and the workspace item, while the empty runtime list still said `暂无审计追踪`. This made the shell feel terminology-heavy after the runtime audit page itself had been simplified.

- [x] **Step 2: Update bilingual copy**

The runtime group is now `运行与审计` / `Runtime & Audit`, the workspace remains `运行审计` / `Runtime Audit`, and the empty trace state now talks about runtime records.

- [x] **Step 3: Verify focused tests and repository gates**

Run i18n focused tests plus `git diff --check`, `make check`, and `make release-check`; then commit and push if clean.

Verified locally before commit:

```bash
pnpm --dir frontend exec node --test tests/i18n.test.mjs
git diff --check
make check
make release-check
```

Committed and pushed `262d0fd fix: clarify runtime audit navigation copy` to `codex/production-readiness-gate`. GitHub PR #74 reported `mergeStateStatus=UNSTABLE` because Backend, Frontend, and PostgreSQL integration checks were in progress immediately after push.

## Task 27: Use Business Pickers In Capability Governance

- [x] **Step 1: Inspect the capability governance grant form**

Browser verification on `http://127.0.0.1:5174/#capabilities` showed the grant-chain form still exposed technical defaults (`default`, `workspace-sandbox`, `user:ops-*`) as ordinary fields. This made the capability governance workspace feel like a demo admin console rather than a production permission-management surface.

- [x] **Step 2: Replace raw scope inputs with business pickers**

Capability governance now uses business-labeled tenant and workspace dropdowns, reuses localized caller/target/capability labels, and defaults the access object to the same support-role selector used by the permission change journey.

- [x] **Step 3: Keep custom subject selectors in advanced settings**

Custom subject selector editing moved into a `capability-grant-advanced` disclosure. The grant-chain mutation payload is unchanged; only the operator-facing form path changed.

- [x] **Step 4: Verify focused tests and browser**

Verified:

```bash
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs
```

Browser verification on `http://127.0.0.1:5174/#capabilities` showed the main grant form as `集团总部 / 客户服务工作区 / 角色 · 客服专员` and no visible `tenantId`, `workspaceId`, `default`, `workspace-sandbox`, or `user:ops-*` in the business path.

- [x] **Step 5: Run repository gates, commit, push, and inspect PR checks**

Run `git diff --check`, `make check`, and `make release-check`; then commit and push if clean.

Verified locally before commit:

```bash
git diff --check
make check
make release-check
```

Committed and pushed `982cb92 fix: use business pickers for capability grants` to `codex/production-readiness-gate`. GitHub PR #74 reported fresh Backend, Frontend, and PostgreSQL integration checks as pending immediately after push.

## Task 29: Freeze Completed Permission Configuration

- [x] **Step 1: Inspect the completed Permission Changes first viewport**

Browser verification on `http://127.0.0.1:5174/#ai-admin` showed the journey was already `可上线`, with primary action `导出证据` and secondary action `查看权限画像`, but the request configuration still looked editable. This made a completed production change feel like a draft editor and created an avoidable operator-risk path.

- [x] **Step 2: Add a read-only review state**

The Permission Changes workspace now treats pending approval, approved approval, applied permissions, and production-ready journeys as frozen request configurations. Tenant, caller, target service, access object, region, permission package, and advanced technical fields become disabled, while the panel displays a concise read-only review notice.

- [x] **Step 3: Make the shared dropdown component truly disableable**

`ApprovalDropdown` now has a `disabled` prop that prevents menu opening, keyboard selection, and option rendering, instead of relying on visual treatment alone.

- [x] **Step 4: Verify focused tests, browser, and repository gates**

Run focused frontend tests, inspect the browser state, then run `git diff --check`, `make check`, and `make release-check`.

Verified:

```bash
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs tests/i18n.test.mjs tests/styleTheme.test.mjs
git diff --check
make check
make release-check
```

Browser verification on `http://127.0.0.1:5174/#ai-admin` walked the local flow from `需要审批` through `提交审批` into `可上线`. The request configuration panel showed `当前配置仅供复核`, all 9 configuration controls were disabled, and the task header promoted `导出证据` as the primary action.

- [x] **Step 5: Commit, push, and inspect PR checks**

Commit and push if clean, then inspect PR #74.

Committed the read-only review state update after focused tests, browser verification, `git diff --check`, `make check`, and `make release-check`.

## Task 30: Rename Locked Request Section To Review

- [x] **Step 1: Inspect the locked first viewport copy**

After Task 29, the locked state worked correctly, but the section title still read like editable request input: `申请信息`, with helper copy telling operators to select tenant, caller, tool service, and access role.

- [x] **Step 2: Switch locked copy to review language**

When `requestFormLocked` is true, the same section now renders as `配置复核` / `Configuration Review`, with helper copy explaining that the operator is reviewing the active tenant, caller, tool service, access role, and permission package.

- [x] **Step 3: Verify focused tests, browser, and repository gates**

Run focused tests and browser verification, then run `git diff --check`, `make check`, and `make release-check`.

Verified:

```bash
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs tests/i18n.test.mjs
git diff --check
make check
make release-check
```

Browser verification on `http://127.0.0.1:5174/#ai-admin` confirmed the locked state shows `配置复核`, the review helper copy, `当前配置仅供复核`, 9 disabled configuration controls, status `可上线`, and primary action `导出证据`.

- [x] **Step 4: Commit, push, and inspect PR checks**

Commit and push if clean, then inspect PR #74.

Committed the locked-state review copy update after focused tests, browser verification, `git diff --check`, `make check`, and `make release-check`.

## Task 31: Add Visible New-Change Exit To Active Review

- [x] **Step 1: Inspect the active locked review state**

The locked review state correctly prevented editing and used review copy, but the visible first viewport did not expose the `新建权限变更` exit. The same action existed lower in the completion card, outside the immediate review context.

- [x] **Step 2: Add the exit only for active locks**

`requestFormActiveLocked` now separates active/go-live locks from approval-frozen locks. The read-only review notice shows `新建权限变更` only when permissions are already applied or production-ready; pending/approved approval locks keep their approval-flow path and do not expose this bypass.

- [x] **Step 3: Verify focused tests, browser, and repository gates**

Run focused tests and browser verification, then run `git diff --check`, `make check`, and `make release-check`.

Verified:

```bash
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs
git diff --check
make check
make release-check
```

Browser verification on `http://127.0.0.1:5174/#ai-admin` confirmed the active locked review state shows `新建权限变更` beside the review notice, keeps `导出证据` as the primary action, and keeps all 9 configuration controls disabled.

- [x] **Step 4: Commit, push, and inspect PR checks**

Commit and push if clean, then inspect PR #74.

Committed the active locked-state new-change exit after focused tests, browser verification, `git diff --check`, `make check`, and `make release-check`.

## Task 32: Isolate New Permission Change Drafts

- [x] **Step 1: Browser-test the new-change exit**

Clicking `新建权限变更` from the active locked review notice kept the UI in `可上线 / 配置复核`, with old completion evidence still visible. The reset cleared local state briefly, but the workbench preview immediately matched the same scope and rehydrated historical approval/application/readiness evidence.

- [x] **Step 2: Add a new-draft boundary**

`aiAdminNewDraftMode` now marks a newly started permission change. While active, the frontend still accepts preview drafts, but ignores historical approval request, application, and production-readiness evidence and suppresses automatic approval-queue loading. Creating a real approval request exits this mode.

- [x] **Step 3: Verify focused tests, browser, and repository gates**

Run focused tests and browser verification, then run `git diff --check`, `make check`, and `make release-check`.

Verified:

```bash
pnpm --dir frontend exec node --test tests/permissionJourneySafety.test.mjs tests/permissionFlowLayout.test.mjs
git diff --check
make check
make release-check
```

Browser verification on `http://127.0.0.1:5174/#ai-admin` confirmed `新建权限变更` now resets to `需要审批 / 申请信息 / 提交审批`, hides the old completion card, and re-enables the 9 configuration controls instead of rehydrating old production-ready evidence.

- [x] **Step 4: Commit, push, and inspect PR checks**

Commit and push if clean, then inspect PR #74.

Committed the new-draft isolation fix after focused tests, browser verification, `git diff --check`, `make check`, and `make release-check`.

## Task 33: Stage The Header Secondary Action

- [x] **Step 1: Inspect the new draft header action**

After new-draft isolation, the reset state correctly showed `需要审批 / 申请信息 / 提交审批`, but the header secondary action still said `查看验收明细`, which pointed operators to a later acceptance phase before approval or apply had happened.

- [x] **Step 2: Derive secondary action by journey phase**

The header secondary action now shows `查看处理流程` before runtime validation, switches to `查看验收明细` once validation/acceptance details are relevant, and keeps `查看权限画像` for completed journeys.

- [x] **Step 3: Verify focused tests, browser, and repository gates**

Run focused tests and browser verification, then run `git diff --check`, `make check`, and `make release-check`.

Verified:

```bash
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs tests/i18n.test.mjs
git diff --check
make check
make release-check
```

Browser verification on `http://127.0.0.1:5174/#ai-admin` confirmed early/approval states show `查看处理流程`, while completed state keeps `导出证据` as primary and `查看权限画像` as the secondary action.

- [x] **Step 4: Commit, push, and inspect PR checks**

Commit and push if clean, then inspect PR #74.

Committed the staged secondary-action copy after focused tests, browser verification, `git diff --check`, `make check`, and `make release-check`.

## Task 34: Re-run Browser Journey Gate After UI Hardening

- [x] **Step 1: Run the publishing-grade browser journey gate**

Run the isolated-port AI Admin browser journey after the read-only review, new-draft isolation, and staged secondary-action updates:

```bash
AGENT_HARBOR_BROWSER_GATE_API_PORT=19090 AGENT_HARBOR_BROWSER_GATE_FRONTEND_PORT=15174 MOCK_MCP_PORT=18787 make ai-admin-browser-journey
```

Result: passed with run id `ai-admin-browser-journey-20260611054116`. The gate covered approval withdrawal, split requester/reviewer admin identities, approval, apply, runtime allow/deny, access profile, health, impact, audit, status check, and production report.

- [x] **Step 2: Record evidence**

Updated `docs/engineering/0.2.0-local-validation-record.md` with the post-UI-hardening browser journey run id and coverage.

- [x] **Step 3: Commit, push, and inspect PR checks**

Commit and push the evidence-only update, then inspect PR #74.

Committed the evidence-only recheck update after `git diff --check`.

## Task 35: Isolate Same-Scope Follow-Up Approval Cycles

- [x] **Step 1: Close the historical-evidence boundary**

After `新建权限变更`, a user can intentionally submit another approval request for the same tenant, caller, tool service, access object, and permission template. That new cycle must not inherit the previous cycle's application or production-readiness evidence while the new approval is still pending.

The workbench preview now suppresses production-readiness lookup whenever the matching approval request is pending, rejected, or withdrawn. Existing completed journeys still show their active production evidence when there is no current blocking approval request, and approved unconsumed requests still proceed to apply preflight.

- [x] **Step 2: Clear stale frontend evidence after creating the new request**

Creating an approval request now immediately selects the new request and clears stale workbench preview, application, application health, impact, preflight, and production-readiness state. The UI therefore stays on the current approval cycle until that cycle is approved, applied, and verified.

- [x] **Step 3: Verify focused tests**

Verified:

```bash
go test ./internal/httpapi -run TestPermissionPackageWorkbenchPreviewSummarizesPrimaryJourney -count=1
pnpm --dir frontend test -- permissionJourneySafety
```

The backend test now covers an already-ready permission journey followed by a second same-scope pending approval request, and asserts that the workbench returns `awaiting_approval` without historical application/readiness evidence.

## Task 36: Upgrade Empty States Toward Production Density

- [x] **Step 1: Replace single-line placeholder empty states**

`EmptyRow` now renders a consistent icon plus title/detail structure instead of plain centered gray text. The shared style uses existing design tokens, a 36px icon column, stable 128px default height, and a compact left-aligned variant for embedded explanation panels.

- [x] **Step 2: Add regression coverage**

`frontend/tests/styleTheme.test.mjs` now asserts the shared empty-state component imports the icon, renders `empty-row-icon` and `empty-row-copy`, and uses tokenized spacing, typography, and compact access-decision alignment.

- [x] **Step 3: Verify focused test, build, and browser**

Verified:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs
pnpm --dir frontend build
```

Browser verification on `http://127.0.0.1:5174/#access` found 3 empty states with icons and structured copy. The regular empty states rendered at `min-height: 128px`; the embedded permission-decision explanation kept its compact `min-height: 72px`.

## Task 37: Separate Neutral And Destructive Table Actions

- [x] **Step 1: Reduce destructive visual noise in dense tables**

Shared `.table-action` buttons now default to a neutral surface treatment for ordinary state changes such as activating or moving an Agent back to draft. Destructive operations such as disabling a route policy or Agent opt into `.table-action.is-danger`.

- [x] **Step 2: Add regression coverage**

`frontend/tests/styleTheme.test.mjs` now asserts route-policy disable and Agent disable use `is-danger`, while Agent active/draft state changes remain neutral.

- [x] **Step 3: Verify focused test and browser**

Verified:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs
```

Browser verification on `http://127.0.0.1:5174/#registry` confirmed Agent table state-change buttons render as neutral white buttons and `禁用` renders with the danger treatment, both at the compact 30px control height.

## Task 38: Hide Technical Field Names From Readiness Copy

- [x] **Step 1: Replace leaked field names**

Permission readiness messages now map missing fields to main-journey business labels: tenant, workspace, caller, and access object. Operators no longer see "Tenant ID", "Workspace ID", "Caller instance", or the raw `subjectSelector` field name when a request is incomplete.

- [x] **Step 2: Add regression coverage**

`frontend/tests/permissionJourneySafety.test.mjs` now asserts `permissionReadinessMessages` maps `tenantId`, `workspaceId`, `callerInstanceId`, and `subjectSelector` through the business-facing labels.

## Task 39: Localize Retry Validation Messages

- [x] **Step 1: Stop leaking retry parser text into panels**

Agent registration and route-policy forms now map retry parser validation messages through i18n before writing them to operator-facing message areas. The parser remains stable, but the UI now displays localized retry-attempt and retry-backoff errors.

- [x] **Step 2: Add regression coverage**

`frontend/tests/i18n.test.mjs` covers the Simplified Chinese retry validation strings. `frontend/tests/permissionJourneySafety.test.mjs` asserts Agent and Route Policy submit handlers call `retryFieldValidationMessage(...)` instead of sending `retry.message` directly to the panel.

## Task 40: Pin Permission-Change Context In Access Profile Handoff

- [x] **Step 1: Keep the completed scope above evidence controls**

Tenant Access Profile now keeps the permission-change handoff as a fixed scope summary across tenant, workspace, caller, target, and capability. The capability is no longer only visible in the filter chips, so operators can verify exactly which completed permission change they are reviewing before touching any controls.

- [x] **Step 2: Reframe filters as optional review controls**

When the access profile is opened from a completed permission change, the filter panel title changes from a generic access-scope query to `Adjust viewing scope` / `调整查看范围`. The explanatory copy makes the current change scope the default and tells operators to adjust filters only when they need a different evidence view.

- [x] **Step 3: Add regression coverage**

`frontend/tests/permissionFlowLayout.test.mjs` asserts the handoff scope includes capability, applies the handoff visual state, and switches the filter title/detail based on `handoffContext`. `frontend/tests/i18n.test.mjs` locks the new Chinese copy.

## Task 41: Keep Completed Process Steps Consistent

- [x] **Step 1: Prefer stronger completed evidence over stale preview text**

The Permission Changes process navigation now normalizes workbench preview steps with the stronger evidence already available in the page. If the journey is go-live ready, approval, apply, validation, and acceptance all render complete details. If local approval, application, or runtime validation evidence is stronger than the preview, those individual steps also render complete.

- [x] **Step 2: Add regression coverage**

`frontend/tests/permissionFlowLayout.test.mjs` now asserts the process step display helpers override stale preview details and statuses with completed local evidence, preventing a ready journey from showing the approval step as pending.

## Task 42: Align Process Step Wording With AI Admin

- [x] **Step 1: Replace requester-oriented step labels**

The first two Permission Changes process steps now read `Configure scope` / `配置范围` and `Approval review` / `审批处理`. This matches the AI Admin mental model better than `Request` / `填写申请`, while keeping the submit button copy as the concrete action.

- [x] **Step 2: Add i18n coverage**

`frontend/tests/i18n.test.mjs` now locks the Simplified Chinese process labels so the main journey keeps admin-oriented wording.

## Task 43: Clarify Process Step Accessible Names

- [x] **Step 1: Add localized step labels for assistive technology**

Permission Changes process step buttons now expose `text.permissionProcessStepAria` as their accessible name. The label includes step number, task label, and current evidence detail, so assistive technology reads a clear sentence instead of concatenating visible text such as `1配置范围已选择...`.

- [x] **Step 2: Add regression coverage**

`frontend/tests/permissionFlowLayout.test.mjs` asserts the process step buttons build their aria label from the localized template, and `frontend/tests/i18n.test.mjs` locks the Simplified Chinese label template.

## Task 44: Soften Reviewer Queue Trace Details

- [x] **Step 1: Rename folded request identifiers as trace details**

Permission Changes reviewer queue rows still retain request, tenant, workspace, caller, and target identifiers for audit and troubleshooting. The collapsed summary now reads `Trace details` / `追溯详情` instead of technical request identifiers, so a completed production journey does not frame raw ids as the operator's active task.

- [x] **Step 2: Add regression coverage**

`frontend/tests/permissionFlowLayout.test.mjs` now asserts reviewer queue details use `text.reviewerQueueTraceDetails`, keeps the Simplified Chinese label locked, and prevents the queue summary from reverting to the generic technical-request wording.

## Task 45: Rename Completed Reviewer Queue as Approval Trace

- [x] **Step 1: Separate active queue wording from completed trace wording**

When a Permission Changes journey is already applied or production-ready, the reviewer queue is read-only evidence. The section title, refresh action, and read-only notice now use `Approval Trace` / `审批追溯`, `Refresh trace` / `刷新追溯`, and approval-record wording, while active in-flight journeys still use the pending approval queue wording.

- [x] **Step 2: Lock the completed-state copy**

`frontend/tests/permissionFlowLayout.test.mjs` asserts the component derives read-only title/action keys from `reviewerQueueReadOnly`, and `frontend/tests/i18n.test.mjs` locks the Simplified Chinese labels.
