import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const app = readFileSync(new URL("../src/ConsoleController.tsx", import.meta.url), "utf8");
const i18n = readFileSync(new URL("../src/i18n.ts", import.meta.url), "utf8");
const baseStyles = readFileSync(new URL("../src/styles.css", import.meta.url), "utf8");
const workbenchStyles = readFileSync(new URL("../src/styles/permission-workbench.css", import.meta.url), "utf8");
const styles = `${baseStyles}\n${workbenchStyles}`;
const workbench = readFileSync(new URL("../src/components/AiAdminPermissionWorkbench.tsx", import.meta.url), "utf8");
const permissionPackages = readFileSync(new URL("../src/permissionPackages.ts", import.meta.url), "utf8");
const permissionWorkbenchPresenters = readFileSync(new URL("../src/permissionWorkbenchPresenters.ts", import.meta.url), "utf8");
const permissionWorkbenchParts = readFileSync(new URL("../src/components/PermissionWorkbenchParts.tsx", import.meta.url), "utf8");
const permissionApprovalDecisionHook = readFileSync(new URL("../src/hooks/usePermissionApprovalDecision.ts", import.meta.url), "utf8");
const dropdown = readFileSync(new URL("../src/components/ApprovalDropdown.tsx", import.meta.url), "utf8");
const technicalId = readFileSync(new URL("../src/components/TechnicalId.tsx", import.meta.url), "utf8");
const accessProfileView = readFileSync(new URL("../src/components/TenantAccessProfileView.tsx", import.meta.url), "utf8");
const adminAccessView = readFileSync(new URL("../src/components/AdminAccessManagementView.tsx", import.meta.url), "utf8");
const adminAccessHook = readFileSync(new URL("../src/hooks/useAdminAccessController.ts", import.meta.url), "utf8");
const capabilityGovernanceView = readFileSync(new URL("../src/components/CapabilityGovernanceView.tsx", import.meta.url), "utf8");
const consoleViews = readFileSync(new URL("../src/components/ConsoleViews.tsx", import.meta.url), "utf8");
const coreJourneyWorkbench = readFileSync(new URL("../src/components/CoreJourneyWorkbench.tsx", import.meta.url), "utf8");
const goLiveAcceptanceOverview = readFileSync(new URL("../src/components/GoLiveAcceptanceOverview.tsx", import.meta.url), "utf8");
const managementForms = readFileSync(new URL("../src/components/ManagementForms.tsx", import.meta.url), "utf8");
const runtimeEvidenceViews = readFileSync(new URL("../src/components/RuntimeEvidenceViews.tsx", import.meta.url), "utf8");
const presenters = readFileSync(new URL("../src/consolePresenters.ts", import.meta.url), "utf8");
const accessProfileHook = readFileSync(new URL("../src/hooks/useAccessProfileController.ts", import.meta.url), "utf8");
const capabilityGovernanceHook = readFileSync(
  new URL("../src/hooks/useCapabilityGovernanceController.ts", import.meta.url),
  "utf8"
);

test("permission request journey renders as one production workspace instead of a demo board", () => {
  assert.match(workbench, /export function AiAdminPermissionWorkbench\(props/);
  assert.match(workbench, /className=\{`approval-studio status-\$\{productionSummary\.status\}`\}/);
  assert.match(workbench, /className="approval-header"/);
  assert.match(workbench, /className="approval-context-bar"/);
  assert.match(workbench, /className="approval-overview"/);
  assert.match(workbench, /className="approval-flow-layout"/);
  assert.match(workbench, /className="approval-request-panel"/);
  assert.match(workbench, /className="approval-process-panel"/);
  assert.match(styles, /\.approval-studio\s*\{[^}]*grid-column:\s*span 12;/s);
  assert.match(styles, /\.approval-overview\s*\{[^}]*padding:\s*0;/s);
  assert.match(styles, /\.approval-task-strip\s*\{[^}]*grid-template-columns:\s*repeat\(3,\s*minmax\(0,\s*1fr\)\);/s);
  assert.match(styles, /\.approval-flow-layout\s*\{[^}]*grid-template-columns:\s*minmax\(0,\s*1fr\)\s*minmax\(340px,\s*400px\);/s);
  assert.match(styles, /\.approval-context-bar\s*\{[^}]*position:\s*static;/s);
  assert.equal(styles.includes("counter-reset: approval-section"), false);
  assert.equal(styles.includes(".approval-section::before"), false);
});

test("permission request workspace does not render the removed product-message UI", () => {
  assert.equal(app.includes("function ProductKeyMessage"), false);
  assert.equal(app.includes("function AiAdminPermissionWorkbenchLegacy"), false);
  assert.equal(app.includes("cockpit-key-message"), false);
  assert.equal(styles.includes(".cockpit-key-message"), false);
  assert.equal(styles.includes(".permission-wizard-card"), false);
  assert.equal(workbench.includes("approval-status-strip"), false);
  assert.equal(workbench.includes("approval-progress"), false);
  assert.equal(workbench.includes("approval-sidebar"), false);
  assert.equal(workbench.includes("ops-"), false);
  assert.equal(styles.includes(".ops-"), false);
});

test("permission request journey separates approval and apply into ordered steps", () => {
  const orderedKeys = [
    "permissionWorkbench.step.request",
    "permissionWorkbench.step.approval",
    "permissionWorkbench.step.apply",
    "permissionWorkbench.step.validation",
    "permissionWorkbench.step.acceptance",
  ];
  let previousIndex = -1;
  for (const key of orderedKeys) {
    const nextIndex = i18n.indexOf(key);
    assert.ok(nextIndex > previousIndex, `${key} should appear after the previous step`);
    previousIndex = nextIndex;
  }
  assert.equal(workbench.includes("section.permissionWizardApprovalApply"), false);
});

test("permission request journey exposes the active step for production operators", () => {
  assert.match(workbench, /currentPermissionRequestWizardStep\(/);
  assert.match(workbench, /aria-current=\{step\.status === "current" \? "step" : undefined\}/);
  assert.match(workbench, /className=\{`approval-process-step status-\$\{step\.status\}`\}/);
  assert.match(styles, /\.approval-process-step\.status-current\s*> span\s*\{/);
  assert.match(styles, /\.approval-process-step\.status-complete\s*> span\s*\{/);
});

test("permission request uses one authoritative journey status", () => {
  assert.match(workbench, /const journeyStatus = resolvePermissionJourneyStatus\(/);
  assert.match(workbench, /journeyStatus\.labelKey/);
  assert.match(workbench, /journeyStatus\.detailKey/);
  assert.match(workbench, /journeyStatus\.tone/);
  assert.match(workbench, /aria-label=\{t\("text\.permissionJourneyStatus"\)\}/);
  assert.doesNotMatch(workbench, /<Badge tone=\{draftStatus\.tone\}>\{t\(draftStatus\.labelKey\)\}<\/Badge>/);
  assert.doesNotMatch(workbench, /const approvalStatusLabel =/);
});

test("permission request process steps navigate to their operator sections", () => {
  assert.match(permissionWorkbenchPresenters, /export function permissionRequestStepSectionId/);
  assert.match(permissionWorkbenchPresenters, /export function permissionRequestStepTarget/);
  assert.doesNotMatch(workbench, /function permissionRequestStepSectionId/);
  assert.doesNotMatch(workbench, /function permissionRequestStepTarget/);
  assert.match(workbench, /function scrollToPermissionRequestStep/);
  assert.match(workbench, /if \(step === "scope" \|\| step === "template"\) \{/);
  assert.match(workbench, /setPermissionDraftSheet\("edit"\);/);
  assert.match(workbench, /document\.getElementById\(permissionRequestStepSectionId\(step\)\)\?\.scrollIntoView/);
  assert.match(permissionWorkbenchPresenters, /if \(step === "request"\) return "scope"/);
  assert.match(permissionWorkbenchPresenters, /if \(step === "validation"\) return "validation"/);
  assert.match(permissionWorkbenchPresenters, /if \(step === "acceptance"\) return "acceptance"/);
  assert.match(permissionWorkbenchParts, /id=\{permissionRequestStepSectionId\("scope"\)\}/);
  assert.match(permissionWorkbenchParts, /id=\{permissionRequestStepSectionId\("template"\)\}/);
  assert.match(workbench, /id=\{permissionRequestStepSectionId\("approval"\)\}/);
  assert.match(workbench, /id=\{permissionRequestStepSectionId\("apply"\)\}/);
  assert.match(workbench, /id=\{permissionRequestStepSectionId\("goLive"\)\}/);
  assert.match(workbench, /id=\{permissionRequestStepSectionId\("validation"\)\}/);
  assert.match(workbench, /id=\{permissionRequestStepSectionId\("acceptance"\)\}/);
  assert.match(workbench, /targetStep: permissionRequestStepTarget\(step\.key\)/);
  assert.match(workbench, /const stepLabel = t\(step\.labelKey\)/);
  assert.match(workbench, /aria-label=\{tx\(t, "text\.permissionProcessStepAria", \{ detail: step\.detail, index: index \+ 1, label: stepLabel \}\)\}/);
  assert.match(workbench, /<button[\s\S]*data-step-target=\{step\.targetStep\}[\s\S]*onClick=\{\(\) => scrollToPermissionRequestStep\(step\.targetStep\)\}/);
  assert.match(styles, /\.approval-process-step\s*\{[^}]*border:\s*0;/s);
  assert.match(styles, /\.approval-process-step:hover\s*\{/);
});

test("permission request process navigation hides request capability counts", () => {
  assert.match(workbench, /count: step\.key === "request" \? undefined : step\.count/);
  assert.match(workbench, /total: step\.key === "request" \? undefined : step\.total/);
  assert.match(workbench, /typeof step\.count === "number" && typeof step\.total === "number"/);
});

test("administrator boundary workspace uses modal actions and never renders key hashes", () => {
  assert.match(app, /const viewCanRenderWithoutConsoleData = activeView\.key === "admin-access";/);
  assert.match(app, /!\s*data && !viewCanRenderWithoutConsoleData/);
  assert.match(adminAccessHook, /forbidden:\s*false/);
  assert.match(adminAccessHook, /error\.adminAccessPlatformRequired/);
  assert.match(adminAccessView, /!controller\.forbidden/);
  assert.match(adminAccessView, /adminAccess\.forbiddenTitle/);
  assert.match(adminAccessView, /className="admin-access-empty-state"/);
  assert.doesNotMatch(adminAccessView, /<td colSpan=\{8\}>/);
  assert.match(styles, /\.admin-access-empty-state,\s*\n\.management-audit-empty-state\s*\{[^}]*min-height:\s*128px;/s);
  assert.match(adminAccessView, /admin-access-modal-backdrop/);
  assert.match(adminAccessView, /controller\.oneTimeKey/);
  assert.match(adminAccessView, /className="primary-button"[\s\S]*t\("adminAccess\.create"\)/);
  assert.match(adminAccessView, /className="primary-button"[\s\S]*t\("adminAccess\.rotate"\)/);
  assert.match(adminAccessView, /className="danger-button"[\s\S]*t\("adminAccess\.disable"\)/);
  assert.doesNotMatch(adminAccessView, /keyHash/);
});

test("permission request process steps prefer completed evidence over stale preview copy", () => {
  assert.match(workbench, /permissionWorkbenchStepDisplayDetailCode\(step,\s*\{[\s\S]*goLiveReady[\s\S]*runtimeValidationReady[\s\S]*\}\)/);
  assert.match(workbench, /permissionWorkbenchStepDisplayStatus\(step,\s*\{[\s\S]*approvalComplete[\s\S]*applicationReady[\s\S]*goLiveReady[\s\S]*runtimeValidationReady[\s\S]*\}\)/);
  assert.match(permissionWorkbenchPresenters, /export function permissionWorkbenchStepDisplayDetailCode/);
  assert.match(permissionWorkbenchPresenters, /export function permissionWorkbenchStepDisplayStatus/);
  assert.doesNotMatch(workbench, /function permissionWorkbenchStepDisplayDetailCode/);
  assert.doesNotMatch(workbench, /function permissionWorkbenchStepDisplayStatus/);
  assert.match(permissionWorkbenchPresenters, /if \(args\.goLiveReady\) \{[\s\S]*if \(step\.key === "approval"\) return args\.approvalRequired \? "approval_approved" : "approval_not_required";[\s\S]*if \(step\.key === "apply"\) return "apply_done";[\s\S]*if \(step\.key === "validation"\) return "validation_ready";[\s\S]*if \(step\.key === "acceptance"\) return "acceptance_ready";[\s\S]*\}/);
  assert.match(permissionWorkbenchPresenters, /if \(args\.goLiveReady\) return "complete";/);
});

test("permission request first viewport prioritizes one task flow", () => {
  const headerStart = workbench.indexOf('<section className="approval-header"');
  const contextStart = workbench.indexOf('<section className="approval-context-bar"', headerStart);
  const overviewStart = workbench.indexOf('<section className="approval-overview"', contextStart);
  const taskStripStart = workbench.indexOf('<div className="approval-task-strip"', overviewStart);
  const flowStart = workbench.indexOf('<div className="approval-flow-layout">', overviewStart);
  assert.ok(headerStart >= 0 && contextStart > headerStart && overviewStart > contextStart);
  assert.match(workbench.slice(headerStart, contextStart), /t\("text\.permissionRequestActions"\)/);
  assert.doesNotMatch(workbench.slice(headerStart, contextStart), /<strong>\{t\(journeyStatus\.nextActionKey\)\}<\/strong>/);
  assert.ok(taskStripStart > overviewStart && flowStart > taskStripStart);
  assert.match(workbench.slice(taskStripStart, flowStart), /journeyStatus\.labelKey/);
  assert.match(workbench.slice(taskStripStart, flowStart), /journeyStatus\.nextActionKey/);
  assert.doesNotMatch(workbench.slice(taskStripStart, flowStart), /plannedObjectCount/);
  assert.match(styles, /\.approval-task-strip article\s*\{/);
  assert.match(styles, /\.approval-process-panel\s*\{[^}]*position:\s*sticky;/s);
});

test("permission request top chrome avoids nested card and table treatment", () => {
  assert.match(workbench, /className=\{`approval-studio status-\$\{productionSummary\.status\}`\}/);
  assert.match(styles, /\.approval-studio\s*\{[^}]*border:\s*0;/s);
  assert.match(styles, /\.approval-studio\s*\{[^}]*background:\s*transparent;/s);
  assert.match(styles, /\.approval-command\s*\{[^}]*border:\s*0;/s);
  assert.match(styles, /\.approval-command\s*\{[^}]*background:\s*transparent;/s);
  assert.match(styles, /\.approval-context-bar\s*\{[^}]*gap:\s*8px;/s);
  assert.match(styles, /\.approval-context-bar\s*\{[^}]*position:\s*static;/s);
  assert.match(styles, /\.approval-context-bar\s*\{[^}]*box-shadow:\s*none;/s);
  assert.match(styles, /\.approval-context-bar div\s*\{[^}]*border:\s*1px solid var\(--line-muted\);/s);
  assert.match(styles, /\.approval-task-strip\s*\{[^}]*gap:\s*8px;/s);
  assert.match(styles, /\.approval-task-strip\s*\{[^}]*border:\s*0;/s);
  assert.match(styles, /\.approval-task-strip article\s*\{[^}]*border:\s*1px solid var\(--line-muted\);/s);
});

test("permission request embeds a concise concept guide without blocking the task flow", () => {
  assert.match(permissionWorkbenchParts, /<details className="approval-concept-guide">/);
  assert.match(permissionWorkbenchParts, /<summary>\{t\("section\.permissionConceptGuide"\)\}<\/summary>/);
  assert.match(permissionWorkbenchParts, /className="approval-concept-grid"/);
  assert.match(permissionWorkbenchParts, /concept\.tenant/);
  assert.match(permissionWorkbenchParts, /concept\.caller/);
  assert.match(permissionWorkbenchParts, /concept\.permissionPackage/);
  assert.match(permissionWorkbenchParts, /concept\.evidence/);
  assert.match(styles, /\.approval-concept-guide\s*\{/);
  assert.match(styles, /\.approval-concept-grid\s*\{/);
  assert.match(i18n, /"section\.permissionConceptGuide": "概念速览"/);
});

test("permission request stays navigable at tablet desktop widths", () => {
  const globalResponsiveIndex = baseStyles.indexOf("@media (max-width: 1120px)");
  const permissionOverrideIndex = baseStyles.lastIndexOf("@media (min-width: 900px) and (max-width: 1120px)");
  assert.ok(globalResponsiveIndex >= 0);
  assert.ok(permissionOverrideIndex > globalResponsiveIndex);
  assert.match(styles, /@media \(max-width: 1120px\)\s*\{[\s\S]*?\.approval-overview\s*\{[\s\S]*?grid-template-columns:\s*minmax\(0,\s*1fr\);/);
  assert.match(styles, /@media \(min-width: 900px\) and \(max-width: 1120px\)\s*\{[\s\S]*?\.approval-flow-layout\s*\{[\s\S]*?grid-template-columns:\s*minmax\(0,\s*1fr\)\s+minmax\(300px,\s*340px\);/);
  assert.match(styles, /@media \(min-width: 900px\) and \(max-width: 1120px\)\s*\{[\s\S]*?\.approval-process-panel\s*\{[\s\S]*?position:\s*sticky;/);
  assert.match(styles, /@media \(max-width: 760px\)\s*\{[\s\S]*?\.approval-context-bar\s*\{[\s\S]*?grid-template-columns:\s*1fr;/);
  assert.match(styles, /@media \(max-width: 760px\)\s*\{[\s\S]*?\.approval-task-strip\s*\{[\s\S]*?grid-template-columns:\s*1fr;/);
});

test("permission request copy avoids repeated step labels", () => {
  assert.doesNotMatch(i18n, /"permissionWorkbench\.detail\.approval_approved": "审批已通过且匹配当前申请。"/);
  assert.doesNotMatch(i18n, /"permissionWorkbench\.detail\.apply_done": "权限已应用。"/);
  assert.doesNotMatch(i18n, /"permissionWorkbench\.detail\.validation_ready": "运行记录已完整。"/);
  assert.doesNotMatch(i18n, /"permissionWorkbench\.detail\.acceptance_ready": "上线就绪检查已完成。"/);
  assert.match(i18n, /"permissionWorkbench\.detail\.approval_approved": "审批已满足。"/);
  assert.match(i18n, /"permissionWorkbench\.detail\.apply_done": "权限已生效。"/);
  assert.match(i18n, /"permissionWorkbench\.detail\.validation_ready": "运行验证已通过。"/);
  assert.match(i18n, /"permissionWorkbench\.detail\.acceptance_ready": "状态检查已通过。"/);
  assert.match(i18n, /"section\.permissionWizardApproval": "审批处理"/);
  assert.match(i18n, /"section\.permissionWizardGoLive": "状态检查"/);
  assert.match(i18n, /"section\.goLiveAcceptance": "上线检查"/);
  assert.doesNotMatch(i18n, /"section\.permissionWizardApproval": "提交审批"/);
  assert.match(styles, /@media \(prefers-reduced-motion: reduce\)\s*\{/);
  assert.match(styles, /\.approval-dropdown-trigger svg\s*\{[^}]*transition:\s*none;/s);
});

test("permission request carries readable context into access profile workspace", () => {
  assert.match(accessProfileView, /handoffContext\?: AccessProfileHandoffContext \| null/);
  assert.match(accessProfileView, /const tenantLabel = handoffContext\?\.tenantName[\s\S]*permissionEntityDisplayName\(profile\.tenant\.name \|\| profile\.tenant\.id, t\)[\s\S]*permissionEntityDisplayName\(scope\.tenantId, t\)/);
  assert.match(accessProfileView, /const workspaceLabel = handoffContext\?\.workspaceName \?\? \(filters\.workspaceId\?\.trim\(\) \|\| t\("form\.workspaceAll"\)\)/);
  assert.match(accessProfileView, /const callerLabel = handoffContext\?\.callerName \?\? \(filters\.callerInstanceId/);
  assert.match(accessProfileView, /const targetLabel = handoffContext\?\.targetName \?\? \(filters\.targetId/);
  assert.match(accessProfileView, /const capabilityLabel = handoffContext\?\.capabilityName \?\? \(selectedCapability/);
  assert.match(accessProfileView, /className="access-handoff-context"/);
  assert.match(accessProfileView, /text\.accessProfileHandoffDetail/);
  assert.match(accessProfileView, /className=\{`access-selected-scope \$\{handoffContext \? "is-handoff" : ""\}`\}/);
  assert.match(accessProfileView, /<span>\{t\("form\.capability"\)\}<\/span>\s*<strong>\{capabilityLabel\}<\/strong>/);
  assert.match(accessProfileView, /handoffContext \? t\("section\.accessProfileAdjustScope"\) : t\("section\.accessProfileFilters"\)/);
  assert.match(accessProfileView, /handoffContext \? t\("text\.accessProfileAdjustScopeDetail"\) : t\("text\.accessProfileFiltersDetail"\)/);
  assert.match(accessProfileHook, /const \[handoffContext, setHandoffContext\] = useState<AccessProfileHandoffContext \| null>\(null\)/);
  assert.match(accessProfileHook, /function clearForPermissionChangeHandoff\(nextFilters: AccessProfileFilters, nextContext: AccessProfileHandoffContext\)/);
  assert.match(accessProfileHook, /setHandoffContext\(nextContext\)/);
  assert.match(app, /accessProfileController\.clearForPermissionChangeHandoff\(\{/);
  assert.match(app, /capabilityName: selectedCapability \? capabilityDisplayName\(selectedCapability, t\) : ""/);
  assert.match(app, /tenantName: permissionTenantPathLabel\(aiAdminForm\.tenantId, tenants, t\)\.primary/);
  assert.match(app, /workspaceName: permissionWorkspaceDisplayName\(aiAdminForm\.workspaceId, agents, t\)/);
  assert.match(app, /handoffContext=\{accessProfileController\.handoffContext\}/);
  assert.match(i18n, /"text\.accessProfileHandoffDetail"/);
  assert.match(i18n, /"section\.accessProfileAdjustScope"/);
});

test("access profile tenant scope prefers business labels over raw tenant ids", () => {
  assert.match(technicalId, /export function TechnicalId/);
  assert.match(workbench, /import \{ TechnicalId \} from "\.\/TechnicalId"/);
  assert.match(accessProfileView, /import \{ TechnicalId \} from "\.\/TechnicalId"/);
  assert.match(accessProfileView, /const tenantNameById = useMemo/);
  assert.match(accessProfileView, /const tenantName = permissionEntityDisplayName\(tenant\.name \|\| tenant\.id, t\)/);
  assert.match(accessProfileView, /tenantLevelLabel\(tenant\.level, t\)/);
  assert.match(accessProfileView, /const parentTenantName = tenant\.parentTenantId \? tenantNameById\.get\(tenant\.parentTenantId\)/);
  assert.match(accessProfileView, /t\("text\.parentTenantOutsideScope"\)/);
  assert.match(i18n, /"text\.parentTenantOutsideScope": "上级租户未展开"/);
  const tenantListStart = accessProfileView.indexOf('<div className="access-tenant-list"');
  const tenantTechnicalStart = accessProfileView.indexOf('<details className="access-tenant-technical"', tenantListStart);
  assert.notEqual(tenantListStart, -1);
  assert.notEqual(tenantTechnicalStart, -1);
  assert.doesNotMatch(accessProfileView.slice(tenantListStart, tenantTechnicalStart), />\{tenant\.id\}</);
  assert.doesNotMatch(accessProfileView.slice(tenantListStart, tenantTechnicalStart), /\$\{tenant\.parentTenantId\}/);
  assert.match(accessProfileView, /<TechnicalId label=\{t\("form\.tenantId"\)\} value=\{tenant\.id\}/);
  assert.match(accessProfileView, /<TechnicalId label=\{t\("text\.parentTenant"\)\} value=\{tenant\.parentTenantId\}/);
  assert.match(styles, /\.access-tenant-technical\s*\{[^}]*grid-column:\s*1 \/ -1;/s);
});

test("access profile query keeps raw identifiers in advanced filters", () => {
  const queryGridStart = accessProfileView.indexOf('<div className="access-query-grid">');
  const advancedFiltersStart = accessProfileView.indexOf('<details className="access-advanced-filters"', queryGridStart);
  const querySummaryStart = accessProfileView.indexOf('<div className="access-query-summary">', advancedFiltersStart);
  assert.notEqual(queryGridStart, -1);
  assert.notEqual(advancedFiltersStart, -1);
  assert.notEqual(querySummaryStart, -1);
  assert.doesNotMatch(accessProfileView.slice(queryGridStart, advancedFiltersStart), /name="tenantId"/);
  assert.doesNotMatch(accessProfileView.slice(queryGridStart, advancedFiltersStart), /name="workspaceId"/);
  assert.doesNotMatch(accessProfileView.slice(queryGridStart, advancedFiltersStart), /name="subjectId"/);
  assert.match(accessProfileView.slice(advancedFiltersStart, querySummaryStart), /name="tenantId"/);
  assert.match(accessProfileView.slice(advancedFiltersStart, querySummaryStart), /name="workspaceId"/);
  assert.match(accessProfileView.slice(advancedFiltersStart, querySummaryStart), /name="subjectId"/);
  assert.match(accessProfileView, /<summary>\{t\("text\.filterSettings"\)\}<\/summary>/);
  assert.match(accessProfileView, /<details className="access-advanced-filters" open=\{!handoffContext\}>/);
  assert.doesNotMatch(accessProfileView, /<span>\{subjectLabel\}<\/span>/);
  assert.match(styles, /\.access-advanced-filters\s*\{/);
});

test("access profile grant chain keeps technical ids in advanced details", () => {
  assert.match(accessProfileView, /import \{ accessSubjectOptionForSelector \} from "\.\.\/accessSubjects"/);
  assert.match(accessProfileView, /const targetName = grant\.target \? permissionEntityDisplayName\(grant\.target\.name, t\) : t\("text\.unknownTarget"\)/);
  assert.match(accessProfileView, /const workspaceLabel = t\("text\.workspaceAssignment"\)/);
  const grantHeaderStart = accessProfileView.indexOf('<div className="access-grant-header">');
  const grantTechnicalStart = accessProfileView.indexOf('<details className="access-grant-technical"', grantHeaderStart);
  assert.notEqual(grantHeaderStart, -1);
  assert.notEqual(grantTechnicalStart, -1);
  assert.doesNotMatch(accessProfileView.slice(grantHeaderStart, grantTechnicalStart), /grant\.tenantEntitlement\.id/);
  assert.doesNotMatch(accessProfileView.slice(grantHeaderStart, grantTechnicalStart), /grant\.tenantEntitlement\.targetId/);
  assert.match(accessProfileView, /<TechnicalId label=\{t\("text\.tenantEntitlement"\)\} value=\{grant\.tenantEntitlement\.id\}/);
  assert.match(accessProfileView, /<TechnicalId label=\{t\("form\.target"\)\} value=\{grant\.tenantEntitlement\.targetId\}/);
  const workspaceMainStart = accessProfileView.indexOf('<div className="access-row-main">');
  const workspaceTechnicalStart = accessProfileView.indexOf('<details className="access-workspace-technical"', workspaceMainStart);
  assert.notEqual(workspaceMainStart, -1);
  assert.notEqual(workspaceTechnicalStart, -1);
  assert.doesNotMatch(accessProfileView.slice(workspaceMainStart, workspaceTechnicalStart), /workspace\.workspaceAssignment\.workspaceId/);
  assert.match(accessProfileView, /<TechnicalId label=\{t\("text\.workspaceAssignment"\)\} value=\{workspace\.workspaceAssignment\.id\}/);
  assert.match(accessProfileView, /<TechnicalId label=\{t\("form\.workspaceId"\)\} value=\{workspace\.workspaceAssignment\.workspaceId\}/);
  assert.match(accessProfileView, /const callerName = instance\.callerInstance[\s\S]*permissionEntityDisplayName\(instance\.callerInstance\.name, t\)[\s\S]*permissionEntityDisplayName\(instance\.instanceAssignment\.callerInstanceId, t\)/);
  assert.match(accessProfileView, /const subjectLabel = subjectSelectorDisplayName\(instance\.instanceAssignment\.subjectSelector, t\)/);
  const instanceMainStart = accessProfileView.indexOf('<div className="access-instance-main">');
  const instanceTechnicalStart = accessProfileView.indexOf('<details className="access-instance-technical"', instanceMainStart);
  assert.notEqual(instanceMainStart, -1);
  assert.notEqual(instanceTechnicalStart, -1);
  assert.doesNotMatch(accessProfileView.slice(instanceMainStart, instanceTechnicalStart), /instance\.instanceAssignment\.subjectSelector/);
  assert.doesNotMatch(accessProfileView.slice(instanceMainStart, instanceTechnicalStart), /instance\.instanceAssignment\.callerInstanceId/);
  assert.match(accessProfileView, /<TechnicalId label=\{t\("text\.instanceAssignment"\)\} value=\{instance\.instanceAssignment\.id\}/);
  assert.match(accessProfileView, /<TechnicalId label=\{t\("form\.subjectSelector"\)\} value=\{instance\.instanceAssignment\.subjectSelector\}/);
  assert.match(accessProfileView, /function subjectSelectorDisplayName/);
  assert.match(accessProfileView, /const traceCallerName = trace\.callerAgentId[\s\S]*permissionEntityDisplayName\(names\[trace\.callerAgentId\] \?\? trace\.callerAgentId, t\)/);
  assert.match(accessProfileView, /const traceTargetName = permissionEntityDisplayName\(names\[trace\.targetAgentId\] \?\? trace\.targetAgentId, t\)/);
  assert.match(accessProfileView, /<strong>\{traceCallerName\} → \{traceTargetName\}<\/strong>/);
  assert.match(accessProfileView, /accessTraceReasonLabel\(trace\.reason, trace\.decision === "allowed" \? "allow" : "deny", t\)/);
  assert.match(runtimeEvidenceViews, /accessTraceReasonLabel\(trace\.reason, trace\.decision === "allowed" \? "allow" : "deny", t\)/);
  assert.match(presenters, /export function accessTraceReasonLabel/);
  assert.match(i18n, /"traceReason\.capabilityAssignmentMatched": "权限分配已命中"/);
  assert.match(i18n, /"text\.customSubjectScope": "自定义主体范围"/);
  assert.match(styles, /\.access-grant-technical\s*\{/);
  assert.match(styles, /\.access-workspace-technical[^{]*\{/);
  assert.match(styles, /\.access-instance-technical[^{]*\{/);
});

test("runtime audit keeps protocol details out of primary trace rows", () => {
  assert.match(runtimeEvidenceViews, /function traceRouteBusinessLabel/);
  assert.match(runtimeEvidenceViews, /className="trace-business-line"/);
  assert.match(runtimeEvidenceViews, /className="trace-technical-details"/);
  assert.match(runtimeEvidenceViews, /const \[traceDetailsExpanded, setTraceDetailsExpanded\] = useState\(false\)/);
  assert.match(runtimeEvidenceViews, /open=\{traceDetailsExpanded\}/);
  assert.match(runtimeEvidenceViews, /traceDetailsExpanded \? t\("action\.collapseTraceDetails"\) : t\("action\.expandTraceDetails"\)/);
  assert.match(runtimeEvidenceViews, /<summary>\{t\("text\.traceDetails"\)\}<\/summary>/);
  assert.match(managementForms, /<summary>\{t\("text\.filterSettings"\)\}<\/summary>/);
  assert.doesNotMatch(managementForms, /<input placeholder="runId"/);
  assert.match(managementForms, /placeholder=\{t\("form\.traceRunPlaceholder"\)\}/);
  const traceRowStart = runtimeEvidenceViews.indexOf('<article className="trace-row"');
  const traceTechnicalStart = runtimeEvidenceViews.indexOf('<details className="trace-technical-details"', traceRowStart);
  assert.notEqual(traceRowStart, -1);
  assert.notEqual(traceTechnicalStart, -1);
  assert.doesNotMatch(runtimeEvidenceViews.slice(traceRowStart, traceTechnicalStart), /trace\.routeType/);
  assert.doesNotMatch(runtimeEvidenceViews.slice(traceRowStart, traceTechnicalStart), /trace\.routeKey/);
  assert.doesNotMatch(runtimeEvidenceViews.slice(traceRowStart, traceTechnicalStart), /trace\.capabilityId/);
  assert.match(presenters, /traceReason\.filteredToolsListByCapabilityAssignments/);
  assert.match(presenters, /traceReason\.capabilityNotApproved/);
  assert.match(i18n, /"traceReason\.filteredToolsListByCapabilityAssignments": "工具列表已按权限收敛"/);
  assert.match(i18n, /"traceReason\.capabilityNotApproved": "能力未审批，已拒绝"/);
  assert.match(styles, /\.trace-technical-details\s*\{/);
});

test("system self-check uses structured configuration and copyable runtime context", () => {
  const panelStart = coreJourneyWorkbench.indexOf("export function CoreJourneyWorkbench");
  const panelEnd = coreJourneyWorkbench.indexOf("function CoreJourneyStepRow", panelStart);
  const panel = coreJourneyWorkbench.slice(panelStart, panelEnd);
  const rowStart = coreJourneyWorkbench.indexOf("function CoreJourneyStepRow");
  const rowEnd = coreJourneyWorkbench.indexOf("function coreJourneyStatusTone", rowStart);
  const row = coreJourneyWorkbench.slice(rowStart, rowEnd);

  assert.match(coreJourneyWorkbench, /ChevronRight/);
  assert.match(panel, /className="core-journey-config"/);
  assert.match(panel, /className="core-journey-config-grid"/);
  assert.match(panel, /className="core-journey-health"/);
  assert.match(panel, /className="core-journey-health-summary"/);
  assert.match(panel, /className="core-journey-task"/);
  assert.match(panel, /className="core-journey-advanced"/);
  assert.match(panel, /className="core-journey-runtime-summary"/);
  assert.match(panel, /className="core-journey-disclosure-action"/);
  assert.match(panel, /t\("action\.viewDetails"\)/);
  assert.match(panel, /className="core-journey-runtime-cards"/);
  assert.match(panel, /className="core-journey-runtime-card"/);
  assert.match(panel, /t\("section\.selfCheckRuntimeScope"\)/);
  assert.match(panel, /t\("section\.selfCheckRuntimeDecision"\)/);
  assert.match(panel, /className="core-journey-runtime-diagnostics"/);
  assert.match(panel, /t\("section\.selfCheckDiagnosticIdentifiers"\)/);
  assert.match(panel, /<div className="core-journey-runtime-diagnostics">[\s\S]*<TechnicalId copyLabel=\{t\("action\.copy"\)\} label=\{t\("detail\.runId"\)\} value=\{config\.runId\} \/>/);
  assert.match(panel, /<div className="core-journey-runtime-diagnostics">[\s\S]*<TechnicalId copyLabel=\{t\("action\.copy"\)\} label=\{t\("form\.tenantId"\)\} value=\{config\.childTenantId\} \/>/);
  assert.doesNotMatch(panel, /<div className="core-journey-meta">/);
  assert.match(coreJourneyWorkbench, /coreJourneyStepDetailLabel/);
  assert.match(row, /className="core-journey-step-detail" title=\{step\.detail\}/);
  assert.doesNotMatch(row, /className="core-journey-step-detail" translate="no"/);
  assert.match(row, /className="core-journey-step-metric"/);
  assert.match(styles, /\.core-journey-health\s*\{[^}]*grid-template-columns:\s*minmax\(0,\s*1fr\)\s*auto;/s);
  assert.match(styles, /\.core-journey-health-summary strong\s*\{[^}]*font-size:\s*16px;/s);
  assert.match(styles, /\.core-journey-score strong\s*\{[^}]*font-size:\s*18px;/s);
  assert.match(styles, /\.core-journey-advanced summary\s*\{/);
  assert.match(styles, /\.core-journey-advanced summary\s*\{[^}]*min-height:\s*46px;/s);
  assert.match(styles, /\.core-journey-runtime-summary summary \.core-journey-disclosure-action,\s*\.core-journey-advanced summary \.core-journey-disclosure-action\s*\{/);
  assert.match(styles, /\.core-journey-runtime-summary\[open\] \.core-journey-disclosure-action svg,\s*\.core-journey-advanced\[open\] \.core-journey-disclosure-action svg\s*\{[^}]*transform:\s*rotate\(90deg\);/s);
  assert.match(styles, /\.core-journey-advanced:not\(\[open\]\)\s*>\s*:not\(summary\)\s*\{[^}]*display:\s*none !important;/s);
  assert.match(styles, /\.core-journey-config-grid\s*\{[^}]*grid-template-columns:\s*repeat\(2,\s*minmax\(0,\s*1fr\)\);/s);
  assert.match(styles, /\.core-journey-runtime-summary\s*\{/);
  assert.match(styles, /\.core-journey-runtime-summary:not\(\[open\]\)\s*>\s*:not\(summary\)\s*\{[^}]*display:\s*none !important;/s);
  assert.match(styles, /\.core-journey-runtime-cards\s*\{[^}]*grid-template-columns:\s*repeat\(2,\s*minmax\(0,\s*1fr\)\);/s);
  assert.match(styles, /\.core-journey-runtime-diagnostics\s*\{[^}]*border:\s*1px dashed var\(--line\);/s);
  assert.match(styles, /\.core-journey-preflight-grid,\s*\.core-journey-steps\s*\{[^}]*gap:\s*8px;[^}]*background:\s*transparent;/s);
  assert.match(styles, /\.core-journey-preflight-row,\s*\.core-journey-step\s*\{[^}]*border:\s*1px solid var\(--line-subtle\);[^}]*border-radius:\s*var\(--radius-control\);/s);
  assert.match(styles, /\.core-journey-preflight-row::before,\s*\.core-journey-step::before\s*\{[^}]*display:\s*none;[^}]*content:\s*none;/s);
  assert.match(styles, /\.core-journey-preflight-row strong,\s*\.core-journey-step strong\s*\{[^}]*font-size:\s*13px;/s);
  assert.match(styles, /\.core-journey-preflight-row \.badge,\s*\.core-journey-step \.badge\s*\{[^}]*font-size:\s*11px;/s);
  assert.match(styles, /\.core-journey-preflight-row \.badge,\s*\.core-journey-step \.badge\s*\{[^}]*line-height:\s*1;/s);
  assert.match(styles, /\.core-journey-step\.status-complete::before\s*\{[^}]*background:\s*transparent;/s);
  assert.match(styles, /\.core-journey-step-metric\s*\{[^}]*font-size:\s*11px;/s);
  assert.match(styles, /\.core-journey-preflight-row span\s*\{[^}]*-webkit-line-clamp:\s*2;/s);
  assert.match(styles, /\.core-journey-step-detail\s*\{/);
});

test("capability names use business labels in primary UI", () => {
  assert.match(presenters, /export function capabilityDisplayName/);
  assert.match(presenters, /export function capabilityKeyDisplayName/);
  assert.match(presenters, /export function capabilitySummaryText/);
  assert.match(presenters, /capability\.\$\{capability\.key\}\.summary/);
  assert.match(presenters, /export function dataScopeValueLabels/);
  assert.match(presenters, /"us-east"/);
  assert.match(presenters, /scope\.region/);
  assert.match(presenters, /"default": t\("text\.defaultTenantName"\)/);
  assert.match(presenters, /"default-bu": t\("demo\.permissionRequestApprovalTeam"\)/);
  assert.match(presenters, /"Default Tenant": t\("text\.defaultTenantName"\)/);
  assert.match(presenters, /"Default Business Unit": t\("demo\.permissionRequestApprovalTeam"\)/);
  assert.match(presenters, /"Default Workspace Team": t\("demo\.permissionRequestApprovalProject"\)/);
  assert.match(presenters, /"Security Reviewer": t\("accessSubject\.securityReviewer\.name"\)/);
  assert.match(permissionWorkbenchPresenters, /"Security Reviewer": t\("accessSubject\.securityReviewer\.name"\)/);
  assert.match(presenters, /normalized\.startsWith\("MCP Capability Caller"\)[\s\S]*demo\.mcpCapabilityCaller/);
  assert.match(presenters, /normalized\.startsWith\("MCP Capability MCP Target"\)[\s\S]*demo\.mcpCapabilityTarget/);
  assert.match(permissionWorkbenchPresenters, /normalized\.startsWith\("MCP Capability Caller"\)[\s\S]*demo\.mcpCapabilityCaller/);
  assert.match(permissionWorkbenchPresenters, /normalized\.startsWith\("MCP Capability MCP Target"\)[\s\S]*demo\.mcpCapabilityTarget/);
  assert.match(presenters, /export function dataScopeText\(scopes\?: DataScope\[\], t\?: Translator\)/);
  assert.match(i18n, /"capability\.search_customer\.name": "查询客户"/);
  assert.match(i18n, /"capability\.update_ticket\.name": "更新工单"/);
  assert.match(i18n, /"capability\.export_contracts\.name": "导出合同"/);
  assert.match(i18n, /"capability\.export_contracts\.summary": "导出合同包；属于高风险能力，需要明确审批。"/);
  assert.match(i18n, /"dataScope\.support": "客服工单数据"/);
  assert.match(i18n, /"dataScope\.us-east": "美东"/);
  assert.match(i18n, /"text\.tenantLevel\.0": "1级租户"/);
  assert.match(i18n, /"text\.defaultTenantName": "集团总部"/);
  assert.match(i18n, /"text\.defaultWorkspaceName": "客户服务工作区"/);
  assert.doesNotMatch(i18n, /"text\.defaultWorkspaceName": "沙箱工作区"/);
  assert.match(app, /import \{[\s\S]*capabilityDisplayName,[\s\S]*\} from "\.\/consolePresenters"/);
  assert.match(capabilityGovernanceHook, /message\.capabilityApproved", \{ name: capabilityDisplayName\(capability, t\) \}/);
  assert.match(app, /permissionPolicyGateMessages,[\s\S]*from "\.\/permissionWorkbenchPresenters"/);
  assert.match(workbench, /permissionPolicyReasonMessage,[\s\S]*from "\.\.\/permissionWorkbenchPresenters"/);
  assert.match(permissionWorkbenchParts, /capabilityDisplayName\(capability, t\)\} · \{t\(`value\.\$\{capability\.action\}`/);
  assert.match(permissionWorkbenchPresenters, /key === "capability"[\s\S]*capabilityKeyDisplayName\(value, t\)/);
  assert.match(accessProfileView, /const capabilityNameById = useMemo/);
  assert.match(accessProfileView, /capabilityNameById\.get\(trace\.capabilityId\) \?\? trace\.capabilityId/);
  assert.match(accessProfileView, /const capabilityName = grant\.capability[\s\S]*capabilityDisplayName\(grant\.capability, t\)/);
  assert.match(accessProfileView, /dataScopeValueLabels\(t\)/);
  assert.match(accessProfileView, /tenantLevelLabel\(tenant\.level, t\)/);
  assert.match(accessProfileView, /permissionEntityDisplayName\(profile\.tenant\.name \|\| profile\.tenant\.id, t\)/);
  assert.match(accessProfileView, /summarizeDataScopes\(trace\.dataScopes, t\("text\.noDataScope"\), dataScopeLabels\)/);
  assert.match(accessProfileView, /summarizeDataScopes\(grant\.effectiveTenantDataScopes, t\("text\.noDataScope"\), dataScopeLabels\)/);
  assert.match(capabilityGovernanceView, /label: capabilityDisplayName\(capability, t\)/);
  assert.match(capabilityGovernanceView, /Object\.fromEntries\(agents\.map\(\(agent\) => \[agent\.id, permissionEntityDisplayName\(agent\.name, t\)\]\)\)/);
  assert.match(capabilityGovernanceView, /label: permissionEntityDisplayName\(target\.name, t\)/);
  assert.match(capabilityGovernanceView, /label: permissionEntityDisplayName\(agent\.name, t\)/);
  assert.match(capabilityGovernanceView, /<strong>\{capabilityDisplayName\(capability, t\)\}<\/strong>/);
  assert.match(capabilityGovernanceView, /capabilitySummaryText\(capability, t\)/);
  assert.match(capabilityGovernanceView, /dataScopeText\(selectedCapability\?\.dataScopes, t\)/);
  assert.match(capabilityGovernanceView, /translatedValue\(t, selectedCapability\.sensitivity\)/);
  assert.match(capabilityGovernanceView, /translatedValue\(t, selectedCapability\.riskLevel\)/);
  assert.match(capabilityGovernanceView, /capability \? capabilityDisplayName\(capability, t\) : entitlement\.capabilityId/);
  assert.match(app, /tenants=\{tenants\}/);
  assert.match(app, /normalizedTenantId === defaultManagementScope\.tenantId[\s\S]*text\.defaultTenantName/);
  assert.match(app, /normalizedWorkspaceId === defaultManagementScope\.workspaceId[\s\S]*text\.defaultWorkspaceName/);
  assert.match(permissionWorkbenchPresenters, /normalizedTenantId === "default"[\s\S]*text\.defaultTenantName/);
  assert.match(permissionWorkbenchPresenters, /normalizedWorkspaceId === defaultWorkspaceId[\s\S]*text\.defaultWorkspaceName/);
  assert.match(capabilityGovernanceView, /tenants: Tenant\[\]/);
  assert.match(capabilityGovernanceView, /tenantNames\.get\(entitlement\.tenantId\) \?\? entitlement\.tenantId/);
  assert.doesNotMatch(capabilityGovernanceView, /\{entitlement\.tenantId\} · \{policyEffectLabel/);
  assert.doesNotMatch(capabilityGovernanceView, /capability\.displayName \|\| capability\.key/);
  assert.doesNotMatch(workbench, /\{capability\.key\} · \{t\(`value\.\$\{capability\.action\}`/);
});

test("permission request acceptance details stay secondary to the main operator task", () => {
  const auditStart = workbench.indexOf('className="approval-evidence"');
  assert.notEqual(auditStart, -1);
  assert.ok(auditStart > workbench.indexOf('<aside className="approval-process-panel"'));
  assert.match(workbench.slice(auditStart), /section\.aiAdminReadiness/);
  assert.match(workbench.slice(auditStart), /section\.permissionProductionReadiness/);
  assert.match(i18n, /"section\.permissionAdvancedChecks": "验收明细"/);
  assert.match(i18n, /"section\.aiAdminApprovalJourney": "运行验证记录"/);
});

test("management audit evidence uses business labels before technical ids", () => {
  const auditTableStart = runtimeEvidenceViews.indexOf("export function ManagementAuditTable");
  const auditTableEnd = runtimeEvidenceViews.indexOf("export function EvidenceTimeline", auditTableStart);
  const auditTable = runtimeEvidenceViews.slice(auditTableStart, auditTableEnd >= 0 ? auditTableEnd : undefined);
  assert.match(runtimeEvidenceViews, /auditActionLabel\(event\.action, t\)/);
  assert.match(runtimeEvidenceViews, /auditResourceTypeLabel\(event\.resourceType, t\)/);
  assert.match(runtimeEvidenceViews, /auditActorLabel\(event\.actor, t\)/);
  assert.match(runtimeEvidenceViews, /auditSummaryLabel\(event\.summary, t\)/);
  assert.match(runtimeEvidenceViews, /className="audit-technical"/);
  assert.match(auditTable, /className="management-audit-empty-state"/);
  assert.doesNotMatch(auditTable, /<td colSpan=\{6\}>/);
  assert.match(auditTable, /<summary>\{t\("text\.auditDetails"\)\}<\/summary>/);
  assert.doesNotMatch(auditTable, /text\.technicalDetails/);
  assert.doesNotMatch(runtimeEvidenceViews, /<span>\{event\.resourceId\}<\/span>/);
  assert.match(i18n, /"auditAction\.permission_package\.applied": "应用权限包"/);
  assert.match(i18n, /"auditActor\.local-dev": "本地开发管理员"/);
  assert.match(i18n, /"text\.auditDetails": "详情"/);
  assert.match(styles, /\.admin-access-empty-state,\s*\n\.management-audit-empty-state\s*\{[^}]*min-height:\s*128px;/s);
  assert.match(styles, /\.audit-technical summary\s*\{/);
  assert.match(styles, /\.badge\s*\{[^}]*text-transform:\s*none;/s);
  assert.doesNotMatch(styles, /\.badge\s*\{[^}]*text-transform:\s*lowercase;/s);
});

test("go-live evidence page starts with acceptance workflow instead of historical runs", () => {
  const evidenceStart = consoleViews.indexOf("export function EvidenceView");
  const cockpitStart = consoleViews.indexOf("export function CockpitView", evidenceStart);
  const evidenceCase = consoleViews.slice(evidenceStart, cockpitStart);
  const evidenceRender = evidenceCase.slice(evidenceCase.indexOf("return ("));
  assert.match(goLiveAcceptanceOverview, /export function GoLiveAcceptanceOverview/);
  assert.match(app, /const goLiveAcceptancePanel =/);
  assert.match(goLiveAcceptanceOverview, /className="go-live-acceptance"/);
  assert.match(goLiveAcceptanceOverview, /buildProductionAcceptanceCenter\(/);
  assert.match(goLiveAcceptanceOverview, /connectionStatus/);
  assert.match(goLiveAcceptanceOverview, /onRunConnectionDiagnostics/);
  assert.match(goLiveAcceptanceOverview, /onRefreshProductionReadiness/);
  assert.match(goLiveAcceptanceOverview, /onExportProductionEvidence/);
  assert.match(goLiveAcceptanceOverview, /onOpenPermissionChange/);
  assert.match(goLiveAcceptanceOverview, /const statusMessage = productionReadinessMessage === t\("message\.permissionProductionReadinessLoaded"\)/);
  assert.match(goLiveAcceptanceOverview, /statusMessage \? <p className="go-live-acceptance-message">\{statusMessage\}<\/p> : null/);
  assert.match(goLiveAcceptanceOverview, /const acceptanceReady = acceptanceCenter\.status === "ready"/);
  assert.match(goLiveAcceptanceOverview, /const primaryAction = renderProductionAcceptanceAction/);
  assert.match(goLiveAcceptanceOverview, /<section className="go-live-acceptance-main">[\s\S]*<div className="go-live-acceptance-decision">[\s\S]*<section className="go-live-acceptance-blockers"[\s\S]*<section className="go-live-acceptance-checks"[\s\S]*<aside className="go-live-acceptance-context"/);
  assert.match(evidenceCase, /goLiveAcceptancePanel/);
  assert.ok(evidenceRender.indexOf("{goLiveAcceptancePanel}") < evidenceRender.indexOf("{evidenceRunsPanel}"));
  assert.match(i18n, /"section\.goLiveAcceptance": "上线检查"/);
  assert.match(i18n, /"productionAcceptance\.title": "上线检查"/);
  assert.match(i18n, /"empty\.evidenceRuns\.detail": "历史自检运行会在这里保留；当前权限变更请以上方上线检查状态为准。"/);
  assert.match(styles, /\.go-live-acceptance\s*\{/);
  assert.match(styles, /\.go-live-acceptance-decision\s*\{[^}]*grid-template-columns:\s*minmax\(0,\s*1fr\)\s*auto;/s);
  assert.match(styles, /\.go-live-acceptance-blockers\s*\{/);
  assert.match(styles, /\.go-live-acceptance-context dl\s*\{[^}]*grid-template-columns:\s*repeat\(4,\s*minmax\(0,\s*1fr\)\);/s);
  assert.match(styles, /\.go-live-step-list\s*\{[^}]*grid-template-columns:\s*repeat\(4,\s*minmax\(0,\s*1fr\)\);/s);
});

test("workspace navigation is reflected in the URL hash", () => {
  assert.match(app, /const \[activeNav, setActiveNav\] = useState<NavKey>\(initialNavKey\)/);
  assert.match(app, /window\.history\.replaceState/);
  assert.match(app, /navHashFor\(activeView\.key\)/);
  assert.match(app, /window\.addEventListener\("hashchange"/);
  assert.match(app, /navKeyFromHash\(window\.location\.hash\)/);
});

test("go-live evidence route loads the current permission change preview", () => {
  assert.match(app, /const shouldLoadAiAdminCatalog =\s*consoleAccessReady && \(activeNav === "ask" \|\| activeNav === "ai-admin" \|\| activeNav === "evidence" \|\| activeNav === "tenants"\)/);
  assert.match(app, /shouldLoadAiAdminCatalog \? undefined : normalizedScope\(scope\)/);
  assert.match(app, /const shouldLoadAiAdminWorkbenchPreview = consoleAccessReady && \(activeNav === "ai-admin" \|\| activeNav === "evidence"\)/);
  assert.match(app, /if \(!shouldLoadAiAdminWorkbenchPreview \|\| !data\?\.loadedFromApi \|\| aiAdminNewDraftMode\)/);
  assert.match(app, /previewPermissionPackageWorkbench\(aiAdminForm, adminKey, controller\.signal\)/);
  assert.match(app, /const goLiveAcceptanceForm = aiAdminServerDraft\?\.input \?\? aiAdminForm/);
  assert.match(app, /draft=\{aiAdminServerDraft\}/);
  assert.match(goLiveAcceptanceOverview, /const acceptanceInput = draft\?\.input \?\? form/);
  assert.doesNotMatch(goLiveAcceptanceOverview, /form\.callerInstanceId \|\| t\("text\.unknownCaller"\)/);
  assert.match(i18n, /"text\.callerPendingSelection": "请选择访问用户"/);
  assert.match(i18n, /"text\.targetPendingSelection": "请选择 MCP 目标"/);
});

test("permission request blocks main actions when sample fallback data is shown", () => {
  assert.match(workbench, /liveDataAvailable: boolean/);
  assert.match(workbench, /const liveDataBlocked = !liveDataAvailable/);
  assert.match(workbench, /className="approval-live-warning"/);
  assert.match(workbench, /message\.fallbackDataModeTitle/);
  assert.match(workbench, /message\.fallbackDataModeDetail/);
  assert.match(workbench, /const permissionRequestBusy =/);
  assert.match(workbench, /disabled=\{liveDataBlocked \|\| permissionRequestBusy/);
  assert.match(workbench, /disabled=\{liveDataBlocked \|\| permissionRequestBusy \|\| !canApply\}/);
  assert.doesNotMatch(workbench, /disabled=\{Boolean\(application\) \|\| liveDataBlocked \|\| permissionRequestBusy \|\| !canApply\}/);
  assert.match(styles, /\.approval-live-warning\s*\{/);
});

test("permission request approval decisions show reviewer context before resolving", () => {
  assert.match(permissionApprovalDecisionHook, /export function usePermissionApprovalDecision/);
  assert.match(workbench, /usePermissionApprovalDecision\(\{/);
  assert.match(workbench, /pendingApprovalDecision/);
  assert.match(workbench, /permissionEntityDisplayName\(approvalReviewer\.trim\(\), t\)/);
  assert.match(workbench, /beginApprovalDecision\("approve"/);
  assert.match(workbench, /beginApprovalDecision\("reject"/);
  assert.match(workbench, /className="approval-reviewer-context"/);
  assert.match(workbench, /text\.approvalReviewerIdentity/);
  assert.match(workbench, /text\.approvalReviewerSeparationDetail/);
  assert.match(workbench, /className="approval-decision-confirmation"/);
  assert.match(workbench, /action\.confirmApprovePermissionRequest/);
  assert.match(workbench, /action\.cancelApprovalDecision/);
  assert.doesNotMatch(workbench, /onClick=\{\(\) => onApproveApprovalRequest\(\)\}/);
  assert.doesNotMatch(workbench, /onClick=\{\(\) => onRejectApprovalRequest\(\)\}/);
  assert.match(styles, /\.approval-reviewer-context\s*\{/);
  assert.match(styles, /\.approval-decision-confirmation\s*\{/);
});

test("permission request rejection requires a reviewer reason", () => {
  assert.match(workbench, /form\.approvalRejectReason/);
  assert.match(permissionApprovalDecisionHook, /pendingApprovalDecision\.comment\.trim\(\)/);
  assert.match(permissionApprovalDecisionHook, /message\.permissionApprovalRejectReasonRequired/);
  assert.match(permissionApprovalDecisionHook, /onRejectApprovalRequest\(pendingApprovalDecision\.requestId, comment\)/);
  assert.match(workbench, /action\.confirmRejectPermissionRequest/);
  assert.match(workbench, /text\.approvalRejectReasonHelp/);
  assert.match(app, /async function rejectAiAdminApprovalRequest\(requestId\?: string, comment\?: string\)/);
  assert.match(app, /const reviewerComment = comment\?\.trim\(\)/);
  assert.match(app, /comment: reviewerComment/);
});

test("permission request can withdraw a pending approval request", () => {
  assert.match(workbench, /onWithdrawApprovalRequest: \(comment\?: string\) => void/);
  assert.match(permissionApprovalDecisionHook, /export type ApprovalDecisionAction = "approve" \| "reject" \| "withdraw"/);
  assert.match(permissionApprovalDecisionHook, /onWithdrawApprovalRequest\(comment\)/);
  assert.match(workbench, /beginApprovalDecision\("withdraw"/);
  assert.match(workbench, /action\.withdrawPermissionRequest/);
  assert.match(workbench, /action\.confirmWithdrawPermissionRequest/);
  assert.match(workbench, /text\.approvalWithdrawHelp/);
  assert.match(workbench, /approvalRequestEffectiveStatus !== "pending" && approvalRequestEffectiveStatus !== "approved"/);
  assert.match(workbench, /permissionApprovalStatusLabel\(approvalRequestEffectiveStatus, t\)/);
  assert.match(i18n, /"status\.approvalWithdrawn"/);
  assert.match(i18n, /"message\.permissionApprovalWithdrawn"/);
  assert.match(app, /async function withdrawAiAdminApprovalRequest\(comment\?: string\)/);
});

test("permission request uses effective approval status for expired requests", () => {
  assert.match(permissionPackages, /export type PermissionPackageApprovalEffectiveStatus = PermissionPackageApprovalStatus \| "expired"/);
  assert.match(permissionPackages, /effectiveStatus\?: PermissionPackageApprovalEffectiveStatus/);
  assert.match(permissionPackages, /isExpired\?: boolean/);
  assert.match(permissionPackages, /function permissionPackageApprovalEffectiveStatus/);
  assert.match(workbench, /const approvalRequestEffectiveStatus = approvalRequest \? permissionPackageApprovalEffectiveStatus\(approvalRequest\) : null/);
  assert.match(workbench, /approvalRequests\.filter\(\(request\) => permissionPackageApprovalEffectiveStatus\(request\) === "pending"\)/);
  assert.match(workbench, /approvalRequestEffectiveStatus === "pending"/);
  assert.match(workbench, /approvalRequestEffectiveStatus === "approved"/);
  assert.match(app, /permissionPackageApprovalEffectiveStatus\(request\) === "approved"/);
  assert.match(app, /permissionPackageApprovalEffectiveStatus\(request\) === "expired"/);
  assert.match(permissionWorkbenchPresenters, /status === "expired"/);
  assert.match(i18n, /"status\.approvalExpired"/);
});

test("permission request primary operations share one busy guard", () => {
  assert.match(workbench, /const permissionRequestBusy =/);
  assert.match(workbench, /approvalJourneyRunning/);
  assert.match(workbench, /approvalReadinessChecking/);
  assert.match(workbench, /applyPreflightLoading/);
  assert.match(workbench, /productionReadinessLoading/);
  assert.match(workbench, /productionEvidenceExporting/);
  assert.match(workbench, /accessDecisionExplanationLoading/);
  assert.match(workbench, /applicationHealthLoading/);
  assert.match(workbench, /applicationImpactLoading/);
  assert.match(workbench, /reviewerQueueLoading/);
  assert.match(workbench, /disabled=\{liveDataBlocked \|\| permissionRequestBusy/);
});

test("permission request shows a concrete completion state with three exits", () => {
  assert.match(workbench, /const productionReady =/);
  assert.match(workbench, /productionReadiness\?\.status === "ready"/);
  assert.match(workbench, /workbenchPreview\?\.summary\.productionReady/);
  assert.match(workbench, /productionSummary\.status === "ready"/);
  assert.match(workbench, /const approvalEffectivelyResolved = !draft\.policyGate\.canApplyDirectly/);
  assert.match(workbench, /approvalRequestEffectiveStatus === "approved" \|\| Boolean\(application\) \|\| goLiveReady/);
  assert.match(workbench, /const runtimeValidationReady = Boolean\(approvalJourneyResult\) \|\| goLiveReady/);
  assert.match(workbench, /const runtimeValidationText = runtimeValidationReady/);
  assert.match(workbench, /<span>\{runtimeValidationText\}<\/span>/);
  assert.match(workbench, /application \? \(/);
  assert.match(workbench, /className="approval-action-status is-complete"/);
  assert.match(workbench, /t\("action\.permissionPackageApplied"\)/);
  assert.doesNotMatch(workbench, /disabled=\{Boolean\(application\) \|\| liveDataBlocked \|\| permissionRequestBusy \|\| !canApply\}/);
  assert.match(workbench, /disabled=\{liveDataBlocked \|\| permissionRequestBusy \|\| !canApply\}/);
  assert.match(styles, /\.approval-action-status\s*\{/);
  assert.match(styles, /\.approval-action-status\.is-complete\s*\{/);
  assert.match(workbench, /const approvalDisplayStatus = approvalEffectivelyResolved/);
  assert.match(workbench, /const showPendingApprovalActions = !application && !goLiveReady && approvalRequestEffectiveStatus === "pending"/);
  assert.match(workbench, /permissionPolicyGateDetailKey\(draft\.policyGate\.canApplyDirectly, approvalDisplayStatus\)/);
  assert.match(workbench, /showPolicyGateReasons && draft\.policyGate\.reasons\.length > 0/);
  assert.doesNotMatch(workbench, /<Badge tone=\{draft\.policyGate\.canApplyDirectly \? "success" : approvalRequest \? approvalStatusTone : "warning"\}>/);
  assert.match(workbench, /className="approval-completion"/);
  assert.match(workbench, /text\.permissionChangeCompleteTitle/);
  assert.match(workbench, /text\.permissionChangeCompleteDetail/);
  assert.match(workbench, /productionReadiness\?\.generatedAt/);
  assert.match(workbench, /action\.exportProductionEvidence/);
  assert.match(workbench, /action\.downloadAcceptanceReport/);
  assert.match(workbench, /action\.openAccessProfile/);
  assert.match(workbench, /action\.startPermissionApproval/);
  const completionActionsStart = workbench.indexOf('<div className="approval-completion-actions">');
  const completionActionsEnd = workbench.indexOf('</div>', completionActionsStart);
  assert.notEqual(completionActionsStart, -1);
  assert.notEqual(completionActionsEnd, -1);
  assert.doesNotMatch(workbench.slice(completionActionsStart, completionActionsEnd), /className="primary-button"/);
  assert.match(workbench, /onOpenAccessProfile/);
  assert.match(workbench, /onStartNewPermissionChange/);
  assert.match(workbench, /const quickSecondaryActionLabel = goLiveReady/);
  assert.match(workbench, /runtimeValidationReady \? t\("action\.openAcceptanceDetails"\) : t\("action\.openProcessDetails"\)/);
  assert.match(workbench, /function scrollToAcceptanceDetails\(\)/);
  assert.match(workbench, /id="permission-request-acceptance-details"/);
  assert.match(workbench, /runtimeValidationReady \? scrollToAcceptanceDetails : \(\) => scrollToPermissionRequestStep\(currentWizardStep\)/);
  assert.doesNotMatch(workbench, /const runQuickSecondaryAction = goLiveReady \? onOpenAccessProfile : onRunApprovalJourney/);
  assert.match(app, /function openAiAdminAccessProfile\(\)/);
  assert.match(app, /setActiveNav\("access"\)/);
  assert.match(app, /function startNewAiAdminPermissionChange\(\)/);
  assert.match(i18n, /"text\.policyGateApprovedDetail": "当前权限变更已记录审批。"/);
  assert.match(i18n, /"action\.permissionPackageApplied": "已应用"/);
  assert.match(i18n, /"action\.downloadAcceptanceReport": "下载验收报告"/);
  assert.match(styles, /\.approval-completion\s*\{/);
  assert.match(styles, /\.approval-completion-actions\s*\{/);
});

test("permission request go-live step presents one guided primary action", () => {
  assert.match(workbench, /if \(journeyStatus\.nextActionKey === "action\.completePermissionRequest"\) \{\s*setPermissionDraftSheet\("edit"\);/);
  assert.match(workbench, /if \(journeyStatus\.nextActionKey === "action\.startPermissionApproval"\) \{\s*startNewPermissionChangeInSheet\(\);/);
  assert.match(workbench, /const goLivePrerequisitesReady = Boolean\(application\) \|\| goLiveReady;/);
  assert.match(workbench, /const goLivePrimaryActionKey = !goLivePrerequisitesReady/);
  assert.match(workbench, /!goLivePrerequisitesReady\s*\?\s*journeyStatus\.nextActionKey/);
  assert.match(workbench, /const goLiveNextActionText = !goLivePrerequisitesReady\s*\?\s*t\(journeyStatus\.nextActionKey\)/);
  assert.match(workbench, /const runtimeValidationText = runtimeValidationReady/);
  assert.match(workbench, /goLivePrerequisitesReady\s*\?\s*t\("text\.runtimeValidationResultPending"\)/);
  assert.match(workbench, /t\("text\.runtimeValidationBlockedDetail"\)/);
  assert.match(workbench, /runtimeValidationReady\s*\?\s*"action\.checkProductionReadiness"/);
  assert.match(workbench, /const runGoLivePrimaryAction = \(\) => \{/);
  assert.match(workbench, /if \(!goLivePrerequisitesReady\) \{\s*runProductionPrimaryAction\(\);/);
  assert.match(workbench, /goLivePrimaryActionKey === "action\.exportProductionEvidence"[\s\S]*onExportProductionEvidence\(\);/);
  assert.match(workbench, /goLivePrimaryActionKey === "action\.checkProductionReadiness"[\s\S]*onRefreshProductionReadiness\(\);/);
  assert.match(workbench, /onClick=\{runGoLivePrimaryAction\}/);
  const goLiveBlockStart = workbench.indexOf('className="approval-process-block approval-go-live-block"');
  const runtimeStart = workbench.indexOf('className="approval-runtime"', goLiveBlockStart);
  const goLiveBlock = workbench.slice(goLiveBlockStart, runtimeStart);
  assert.match(goLiveBlock, /goLivePrimaryActionLabel/);
  assert.doesNotMatch(goLiveBlock, /onClick=\{onRunApprovalJourney\}[\s\S]*onClick=\{onRefreshProductionReadiness\}/);
});

test("permission request advanced messages suppress successful load noise", () => {
  assert.match(permissionWorkbenchPresenters, /export function shouldShowAdvancedStatusMessage/);
  assert.doesNotMatch(workbench, /function shouldShowAdvancedStatusMessage/);
  assert.match(workbench, /approvalReadinessMessage && shouldShowAdvancedStatusMessage\(approvalReadinessMessageTone\)/);
  assert.match(workbench, /applyPreflightMessage && shouldShowAdvancedStatusMessage\(applyPreflightMessageTone\)/);
  assert.match(workbench, /productionReadinessMessage && shouldShowAdvancedStatusMessage\(productionReadinessMessageTone\)/);
  assert.match(workbench, /applicationHealthMessage && shouldShowAdvancedStatusMessage\(applicationHealthMessageTone\)/);
  assert.match(workbench, /applicationImpactMessage && shouldShowAdvancedStatusMessage\(applicationImpactMessageTone\)/);
  assert.match(workbench, /accessDecisionExplanationMessage && shouldShowAdvancedStatusMessage\(accessDecisionExplanationMessageTone\)/);
  assert.doesNotMatch(workbench, /approvalReadinessMessage \? <span/);
  assert.doesNotMatch(workbench, /productionReadinessMessage \? <span/);
  assert.doesNotMatch(workbench, /applicationHealthMessage \? <span/);
});

test("permission request hides raw workspace identifiers from the primary path", () => {
  const workspaceLabelStart = permissionWorkbenchParts.indexOf('approval-readonly-field');
  const technicalDetailsStart = permissionWorkbenchParts.indexOf('<details className="approval-details">');
  assert.notEqual(workspaceLabelStart, -1);
  assert.notEqual(technicalDetailsStart, -1);
  assert.ok(workspaceLabelStart < technicalDetailsStart);
  assert.match(permissionWorkbenchParts.slice(workspaceLabelStart, technicalDetailsStart), /workspaceName/);
  assert.doesNotMatch(permissionWorkbenchParts.slice(workspaceLabelStart, technicalDetailsStart), /form\.workspaceId/);
  assert.match(permissionWorkbenchParts, /<summary>\{t\("text\.technicalOverrides"\)\}<\/summary>/);
  assert.match(styles, /\.approval-details:not\(\[open\]\) > :not\(summary\)\s*\{/);
});

test("permission reviewer queue uses business labels before technical identifiers", () => {
  assert.match(permissionWorkbenchPresenters, /export function permissionApprovalRequestBusinessLabel/);
  assert.doesNotMatch(workbench, /function permissionApprovalRequestBusinessLabel/);
  assert.match(workbench, /import \{ TechnicalId \} from "\.\/TechnicalId"/);
  assert.match(technicalId, /export function TechnicalId/);
  assert.match(workbench, /className="approval-review-row-main"/);
  assert.match(workbench, /className="approval-review-row-meta"/);
  assert.match(workbench, /className="approval-review-row-technical"/);
  const queueStart = workbench.indexOf('<section className="approval-reviewer-queue"');
  const advancedStart = workbench.indexOf('<details className="approval-details"', queueStart);
  assert.notEqual(queueStart, -1);
  assert.notEqual(advancedStart, -1);
  assert.doesNotMatch(workbench.slice(queueStart, advancedStart), /permissionPackageApprovalRouteLabel\(request\)/);
  assert.doesNotMatch(workbench.slice(queueStart, advancedStart), /request\.tenantId/);
  assert.doesNotMatch(workbench.slice(queueStart, advancedStart), /request\.callerInstanceId/);
  assert.match(workbench, /<summary>\{t\("text\.reviewerQueueTraceDetails"\)\}<\/summary>/);
  assert.match(i18n, /"text\.reviewerQueueTraceDetails": "追溯详情"/);
  assert.match(i18n, /"text\.technicalOverrides": "技术覆盖"/);
  assert.match(i18n, /"text\.filterSettings": "筛选条件"/);
  assert.match(i18n, /"text\.traceDetails": "追踪详情"/);
  assert.doesNotMatch(workbench, /<summary>\{t\("text\.technicalRequestIds"\)\}<\/summary>/);
  assert.match(styles, /\.approval-review-row-main\s*\{/);
  assert.match(styles, /\.approval-review-row-meta\s*\{/);
});

test("completed permission journeys make reviewer queue read-only", () => {
  assert.match(workbench, /const reviewerQueueReadOnly = Boolean\(application\) \|\| goLiveReady/);
  assert.match(workbench, /const reviewerQueueTitleKey = reviewerQueueReadOnly \? "section\.permissionApprovalTrace" : "section\.permissionReviewerQueue"/);
  assert.match(workbench, /const reviewerQueueRefreshKey = reviewerQueueReadOnly \? "action\.refreshApprovalTrace" : "action\.refreshReviewerQueue"/);
  assert.match(workbench, /<strong>\{t\(reviewerQueueTitleKey\)\}<\/strong>/);
  assert.match(workbench, /reviewerQueueLoading \? t\("action\.loading"\) : t\(reviewerQueueRefreshKey\)/);
  assert.match(workbench, /reviewerQueueReadOnly \? <span className="approval-inline-message status-info">\{t\("text\.reviewerQueueReadOnlyDetail"\)\}<\/span> : null/);
  assert.match(workbench, /reviewerQueueReadOnly \? \(/);
  assert.match(workbench, /className="approval-review-row-state"/);
  assert.match(workbench, /t\("text\.reviewerQueueReadOnlyAction"\)/);
  assert.match(i18n, /"section\.permissionApprovalTrace": "审批追溯"/);
  assert.match(i18n, /"action\.refreshApprovalTrace": "刷新追溯"/);
  assert.match(i18n, /"text\.reviewerQueueReadOnlyDetail": "权限已经生效，审批记录仅用于追溯。"/);
  assert.match(i18n, /"text\.reviewerQueueReadOnlyAction": "只读"/);
  assert.match(styles, /\.approval-review-row-state\s*\{/);
});

test("permission request keeps tenant workspace and caller context visible", () => {
  const contextStart = workbench.indexOf('<section className="approval-context-bar"');
  const overviewStart = workbench.indexOf('<section className="approval-overview"', contextStart);
  const context = workbench.slice(contextStart, overviewStart);
  assert.notEqual(contextStart, -1);
  assert.notEqual(overviewStart, -1);
  assert.ok(contextStart < overviewStart);
  assert.match(context, /text\.currentWorkspaceContext/);
  assert.match(context, /form\.businessTenant/);
  assert.match(context, /form\.businessWorkspace/);
  assert.match(context, /form\.businessCaller/);
  assert.match(context, /tenantPath\.primary/);
  assert.match(context, /workspaceName/);
  assert.match(context, /callerName/);
  assert.doesNotMatch(context, /form\.workspaceId/);
  assert.doesNotMatch(context, /callerInstanceId/);
});

test("permission request overview keeps context in one authoritative bar", () => {
  const contextStart = workbench.indexOf('<section className="approval-context-bar"');
  const overviewStart = workbench.indexOf('<section className="approval-overview"', contextStart);
  const flowStart = workbench.indexOf('<div className="approval-flow-layout">', overviewStart);
  const context = workbench.slice(contextStart, overviewStart);
  const overview = workbench.slice(overviewStart, flowStart);

  assert.notEqual(contextStart, -1);
  assert.notEqual(overviewStart, -1);
  assert.notEqual(flowStart, -1);
  assert.match(context, /form\.businessTenant/);
  assert.match(context, /form\.businessWorkspace/);
  assert.match(context, /form\.businessCaller/);
  assert.match(overview, /approval-task-strip/);
  assert.doesNotMatch(overview, /form\.businessTenant/);
  assert.doesNotMatch(overview, /tenantPath\.primary/);
  assert.doesNotMatch(overview, /workspaceName/);
  assert.doesNotMatch(overview, /callerName/);
});

test("permission request core selectors avoid forced ellipsis at review widths", () => {
  assert.match(permissionWorkbenchParts, /className="approval-field is-wide"/);
  assert.match(permissionWorkbenchParts, /className="approval-readonly-field is-wide"/);
  assert.match(permissionWorkbenchParts, /className="approval-field approval-subject-field is-wide"/);
  assert.match(permissionWorkbenchParts, /className="approval-select is-wide"/);
  assert.match(styles, /\.approval-form-grid \.approval-dropdown-trigger span\s*\{[^}]*white-space:\s*normal;/s);
  assert.match(styles, /\.approval-form-grid \.approval-dropdown-trigger span\s*\{[^}]*text-overflow:\s*clip;/s);
  assert.match(styles, /@media \(max-width: 1120px\)\s*\{[\s\S]*\.approval-flow-layout,\s*[\s\S]*\.approval-form-grid\s*\{[^}]*grid-template-columns:\s*1fr;/s);
});

test("permission request dropdowns use deduplicated business labels", () => {
  assert.match(permissionWorkbenchPresenters, /export function uniquePermissionEntityOptions/);
  assert.doesNotMatch(workbench, /function uniquePermissionEntityOptions/);
  assert.match(workbench, /const tenantOptions = uniquePermissionEntityOptions/);
  assert.match(workbench, /const callerOptions = uniquePermissionEntityOptions/);
  assert.match(workbench, /const targetOptions = uniquePermissionEntityOptions/);
  assert.match(workbench, /tenantDropdownOptions/);
  assert.match(workbench, /callerDropdownOptions/);
  assert.match(workbench, /targetDropdownOptions/);
});

test("permission request primary path avoids native select menus", () => {
  const formStart = permissionWorkbenchParts.indexOf('<div className="approval-form-grid">');
  const formEnd = permissionWorkbenchParts.indexOf('<div className="approval-package-preview"', formStart);
  const primaryForm = permissionWorkbenchParts.slice(formStart, formEnd);
  assert.notEqual(formStart, -1);
  assert.notEqual(formEnd, -1);
  assert.match(dropdown, /export function ApprovalDropdown/);
  assert.match(dropdown, /label: string/);
  assert.match(dropdown, /aria-labelledby=\{labelId\}/);
  assert.match(dropdown, /aria-activedescendant=\{activeOptionId\}/);
  assert.match(primaryForm, /<ApprovalDropdown/);
  assert.doesNotMatch(primaryForm, /<select/);
});

test("permission change draft editor opens from a command sheet", () => {
  assert.match(workbench, /useState<"closed" \| "edit">\("closed"\)/);
  assert.match(workbench, /<PermissionChangeDraftSheet/);
  assert.match(workbench, /isOpen=\{permissionDraftSheet === "edit"\}/);
  assert.match(workbench, /onOpenDraftSheet=\{\(\) => setPermissionDraftSheet\("edit"\)\}/);
  assert.match(workbench, /onClose=\{\(\) => setPermissionDraftSheet\("closed"\)\}/);
  assert.doesNotMatch(workbench, /className=\{`approval-section approval-request-form-section/);
});

test("permission request freezes configuration after approval or apply", () => {
  const formStart = permissionWorkbenchParts.indexOf('<div className="approval-form-grid">');
  const formEnd = permissionWorkbenchParts.indexOf('<div className="approval-package-preview"', formStart);
  const primaryForm = permissionWorkbenchParts.slice(formStart, formEnd);
  const advancedStart = permissionWorkbenchParts.indexOf('<details className="approval-details">');
  const advancedEnd = permissionWorkbenchParts.indexOf('</details>', advancedStart);
  const advancedForm = permissionWorkbenchParts.slice(advancedStart, advancedEnd);

  assert.match(workbench, /const requestFormLocked = Boolean\(application\)/);
  assert.match(workbench, /approvalRequestEffectiveStatus === "pending"/);
  assert.match(workbench, /approvalRequestEffectiveStatus === "approved"/);
  assert.match(workbench, /const requestFormActiveLocked = Boolean\(application\) \|\| goLiveReady/);
  assert.match(workbench, /const requestFormLockedDetailKey = requestFormActiveLocked\s*\?\s*"text\.permissionRequestLockedActiveDetail"\s*:\s*"text\.permissionRequestLockedApprovalDetail"/);
  assert.match(workbench, /const requestFormTitleKey = requestFormLocked \? "section\.permissionRequestReview" : "section\.permissionRequestForm"/);
  assert.match(workbench, /const requestFormHelpKey = requestFormLocked \? "text\.permissionRequestReviewHelp" : "text\.permissionRequestScopeHelp"/);
  assert.match(workbench, /isLocked=\{requestFormLocked\}/);
  assert.match(workbench, /isActiveLocked=\{requestFormActiveLocked\}/);
  assert.match(workbench, /requestFormTitleKey=\{requestFormTitleKey\}/);
  assert.match(workbench, /requestFormHelpKey=\{requestFormHelpKey\}/);
  assert.match(permissionWorkbenchParts, /text\.permissionRequestLockedTitle/);
  assert.match(permissionWorkbenchParts, /isActiveLocked \? \(/);
  assert.match(permissionWorkbenchParts, /onClick=\{onStartNewPermissionChange\}/);
  assert.match(primaryForm, /disabled=\{isLocked\}/);
  assert.match(advancedForm, /disabled=\{isLocked\}/);
  assert.match(dropdown, /disabled\?: boolean/);
  assert.match(dropdown, /disabled=\{disabled\}/);
  assert.match(dropdown, /const menuOpen = open && !disabled/);
  assert.match(styles, /\.approval-lock-notice\s*\{/);
  assert.match(styles, /\.approval-lock-notice \.secondary-button\s*\{/);
  assert.match(styles, /@media \(max-width: 720px\)\s*\{[\s\S]*\.approval-lock-notice\s*\{[^}]*flex-direction:\s*column;/s);
  assert.match(styles, /\.approval-request textarea:disabled,/);
  assert.match(styles, /\.approval-dropdown\.is-disabled \.approval-dropdown-trigger:hover/);
});

test("permission request chooses access objects instead of raw subject selectors", () => {
  const formStart = permissionWorkbenchParts.indexOf('<div className="approval-form-grid">');
  const formEnd = permissionWorkbenchParts.indexOf('<div className="approval-package-preview"', formStart);
  const primaryForm = permissionWorkbenchParts.slice(formStart, formEnd);
  const advancedStart = permissionWorkbenchParts.indexOf('<details className="approval-details">');
  const advancedEnd = permissionWorkbenchParts.indexOf('</details>', advancedStart);
  const advancedForm = permissionWorkbenchParts.slice(advancedStart, advancedEnd);

  assert.match(app, /fetchPermissionPackageAccessSubjects/);
  assert.match(app, /aiAdminAccessSubjects/);
  assert.match(workbench, /accessSubjectCatalog = normalizeAccessSubjectOptions/);
  assert.match(workbench, /selectedAccessSubject = accessSubjectOptionForSelectorFrom/);
  assert.match(primaryForm, /form\.accessSubject/);
  assert.match(primaryForm, /accessSubjectDropdownOptions/);
  assert.doesNotMatch(primaryForm, /form\.subjectSelector/);
  assert.match(advancedForm, /form\.subjectSelector/);
  assert.match(advancedForm, /text\.subjectSelectorAdvancedHelp/);
});

test("access profile and capability governance are split from the app shell", () => {
  assert.match(app, /import \{ TenantAccessProfileView \} from "\.\/components\/TenantAccessProfileView"/);
  assert.match(app, /import type \{ CapabilityGrantForm \} from "\.\/components\/CapabilityGovernanceView"/);
  assert.match(app, /const CapabilityGovernanceView = lazy\(\(\) => import\("\.\/components\/CapabilityGovernanceView"\)/);
  assert.match(capabilityGovernanceHook, /import type \{ CapabilityGrantForm \} from "\.\.\/components\/CapabilityGovernanceView"/);
  assert.doesNotMatch(app, /function TenantAccessProfileView\(/);
  assert.doesNotMatch(app, /function CapabilityGovernanceView\(/);
  assert.match(accessProfileView, /export function TenantAccessProfileView/);
  assert.match(capabilityGovernanceView, /export function CapabilityGovernanceView/);
  assert.match(presenters, /export function permissionEntityDisplayName/);
  assert.match(presenters, /export function capabilityStatusTone/);
});

test("capability governance uses business pickers instead of native select menus", () => {
  assert.match(capabilityGovernanceView, /accessSubjectOptions/);
  assert.match(capabilityGovernanceView, /accessSubjectDropdownOptions/);
  assert.match(capabilityGovernanceView, /const tenantOptions =/);
  assert.match(capabilityGovernanceView, /const workspaceOptions =/);
  assert.match(capabilityGovernanceView, /<ApprovalDropdown/);
  assert.doesNotMatch(capabilityGovernanceView, /<select/);
  assert.doesNotMatch(capabilityGovernanceView, /<input required value=\{form\.tenantId\}/);
  assert.doesNotMatch(capabilityGovernanceView, /<input required value=\{form\.workspaceId\}/);
  assert.match(capabilityGovernanceView, /selectedAccessSubject\.id === customAccessSubjectOption\.id/);
  assert.match(capabilityGovernanceView, /<details className="capability-grant-advanced"/);
  assert.match(capabilityGovernanceHook, /subjectSelector: "user:support-\*"/);
  assert.doesNotMatch(app, /subjectSelector: "user:ops-\*"/);
});
