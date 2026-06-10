import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const app = readFileSync(new URL("../src/App.tsx", import.meta.url), "utf8");
const i18n = readFileSync(new URL("../src/i18n.ts", import.meta.url), "utf8");
const baseStyles = readFileSync(new URL("../src/styles.css", import.meta.url), "utf8");
const workbenchStyles = readFileSync(new URL("../src/styles/permission-workbench.css", import.meta.url), "utf8");
const styles = `${baseStyles}\n${workbenchStyles}`;
const workbench = readFileSync(new URL("../src/components/AiAdminPermissionWorkbench.tsx", import.meta.url), "utf8");
const dropdown = readFileSync(new URL("../src/components/ApprovalDropdown.tsx", import.meta.url), "utf8");
const technicalId = readFileSync(new URL("../src/components/TechnicalId.tsx", import.meta.url), "utf8");
const accessProfileView = readFileSync(new URL("../src/components/TenantAccessProfileView.tsx", import.meta.url), "utf8");
const capabilityGovernanceView = readFileSync(new URL("../src/components/CapabilityGovernanceView.tsx", import.meta.url), "utf8");
const presenters = readFileSync(new URL("../src/consolePresenters.ts", import.meta.url), "utf8");

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
  assert.match(styles, /\.approval-context-bar\s*\{[^}]*position:\s*sticky;/s);
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
  assert.match(workbench, /function permissionRequestStepSectionId/);
  assert.match(workbench, /function permissionRequestStepTarget/);
  assert.match(workbench, /function scrollToPermissionRequestStep/);
  assert.match(workbench, /document\.getElementById\(permissionRequestStepSectionId\(step\)\)\?\.scrollIntoView/);
  assert.match(workbench, /if \(step === "request"\) return "scope"/);
  assert.match(workbench, /if \(step === "validation"\) return "validation"/);
  assert.match(workbench, /if \(step === "acceptance"\) return "acceptance"/);
  assert.match(workbench, /id=\{permissionRequestStepSectionId\("scope"\)\}/);
  assert.match(workbench, /id=\{permissionRequestStepSectionId\("template"\)\}/);
  assert.match(workbench, /id=\{permissionRequestStepSectionId\("approval"\)\}/);
  assert.match(workbench, /id=\{permissionRequestStepSectionId\("apply"\)\}/);
  assert.match(workbench, /id=\{permissionRequestStepSectionId\("goLive"\)\}/);
  assert.match(workbench, /id=\{permissionRequestStepSectionId\("validation"\)\}/);
  assert.match(workbench, /id=\{permissionRequestStepSectionId\("acceptance"\)\}/);
  assert.match(workbench, /targetStep: permissionRequestStepTarget\(step\.key\)/);
  assert.match(workbench, /<button[\s\S]*data-step-target=\{step\.targetStep\}[\s\S]*onClick=\{\(\) => scrollToPermissionRequestStep\(step\.targetStep\)\}/);
  assert.match(styles, /\.approval-process-step\s*\{[^}]*border:\s*0;/s);
  assert.match(styles, /\.approval-process-step:hover\s*\{/);
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
  assert.doesNotMatch(i18n, /"permissionWorkbench\.detail\.validation_ready": "运行证据已完整。"/);
  assert.doesNotMatch(i18n, /"permissionWorkbench\.detail\.acceptance_ready": "上线就绪检查已完成。"/);
  assert.match(i18n, /"permissionWorkbench\.detail\.approval_approved": "已通过，等待应用。"/);
  assert.match(i18n, /"permissionWorkbench\.detail\.apply_done": "已生效，等待验证。"/);
  assert.match(i18n, /"permissionWorkbench\.detail\.validation_ready": "通过，等待就绪检查。"/);
  assert.match(i18n, /"permissionWorkbench\.detail\.acceptance_ready": "证据已完成。"/);
  assert.match(i18n, /"section\.permissionWizardApproval": "审批处理"/);
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
  assert.match(accessProfileView, /className="access-handoff-context"/);
  assert.match(accessProfileView, /text\.accessProfileHandoffDetail/);
  assert.match(app, /const \[accessProfileHandoffContext, setAccessProfileHandoffContext\] = useState<AccessProfileHandoffContext \| null>\(null\)/);
  assert.match(app, /setAccessProfileHandoffContext\(\{/);
  assert.match(app, /capabilityName: selectedCapability \? capabilityDisplayName\(selectedCapability, t\) : ""/);
  assert.match(app, /tenantName: permissionTenantPathLabel\(aiAdminForm\.tenantId, tenants, t\)\.primary/);
  assert.match(app, /workspaceName: permissionWorkspaceDisplayName\(aiAdminForm\.workspaceId, agents, t\)/);
  assert.match(app, /handoffContext=\{accessProfileHandoffContext\}/);
  assert.match(i18n, /"text\.accessProfileHandoffDetail"/);
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
  assert.match(accessProfileView, /<details className="access-advanced-filters" open=\{!handoffContext\}>/);
  assert.doesNotMatch(accessProfileView, /<span>\{subjectLabel\}<\/span>/);
  assert.match(styles, /\.access-advanced-filters\s*\{/);
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
  assert.match(workbench, /"Security Reviewer": t\("accessSubject\.securityReviewer\.name"\)/);
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
  assert.match(app, /message\.capabilityApproved", \{ name: capabilityDisplayName\(capability, t\) \}/);
  assert.match(app, /key === "capability"[\s\S]*capabilityKeyDisplayName\(value, t\)/);
  assert.match(workbench, /capabilityDisplayName\(capability, t\)\} · \{t\(`value\.\$\{capability\.action\}`/);
  assert.match(workbench, /key === "capability"[\s\S]*capabilityKeyDisplayName\(value, t\)/);
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
  assert.match(workbench, /normalizedTenantId === "default"[\s\S]*text\.defaultTenantName/);
  assert.match(workbench, /normalizedWorkspaceId === defaultWorkspaceId[\s\S]*text\.defaultWorkspaceName/);
  assert.match(capabilityGovernanceView, /tenants: Tenant\[\]/);
  assert.match(capabilityGovernanceView, /tenantNames\.get\(entitlement\.tenantId\) \?\? entitlement\.tenantId/);
  assert.doesNotMatch(capabilityGovernanceView, /\{entitlement\.tenantId\} · \{policyEffectLabel/);
  assert.doesNotMatch(capabilityGovernanceView, /capability\.displayName \|\| capability\.key/);
  assert.doesNotMatch(workbench, /\{capability\.key\} · \{t\(`value\.\$\{capability\.action\}`/);
});

test("permission request evidence is secondary to the main operator task", () => {
  const auditStart = workbench.indexOf('<details className="approval-evidence">');
  assert.notEqual(auditStart, -1);
  assert.ok(auditStart > workbench.indexOf('<aside className="approval-process-panel"'));
  assert.match(workbench.slice(auditStart), /section\.aiAdminReadiness/);
  assert.match(workbench.slice(auditStart), /section\.permissionProductionReadiness/);
});

test("management audit evidence uses business labels before technical ids", () => {
  assert.match(app, /auditActionLabel\(event\.action, t\)/);
  assert.match(app, /auditResourceTypeLabel\(event\.resourceType, t\)/);
  assert.match(app, /auditActorLabel\(event\.actor, t\)/);
  assert.match(app, /auditSummaryLabel\(event\.summary, t\)/);
  assert.match(app, /className="audit-technical"/);
  assert.doesNotMatch(app, /<span>\{event\.resourceId\}<\/span>/);
  assert.match(i18n, /"auditAction\.permission_package\.applied": "应用权限包"/);
  assert.match(i18n, /"auditActor\.local-dev": "本地管理员"/);
  assert.match(styles, /\.audit-technical summary\s*\{/);
});

test("go-live evidence page starts with acceptance workflow instead of historical runs", () => {
  const viewSwitch = app.slice(app.indexOf("switch (activeView.key)"));
  const evidenceCase = viewSwitch.slice(viewSwitch.indexOf('case "evidence":'), viewSwitch.indexOf('case "cockpit":'));
  assert.match(app, /function GoLiveAcceptanceOverview/);
  assert.match(app, /const goLiveAcceptancePanel =/);
  assert.match(app, /className="go-live-acceptance"/);
  assert.match(app, /productionReadinessStatusLabel\(productionReadiness\?\.status, t\)/);
  assert.match(app, /onRefreshProductionReadiness/);
  assert.match(app, /onExportProductionEvidence/);
  assert.match(app, /onOpenPermissionChange/);
  assert.match(evidenceCase, /goLiveAcceptancePanel/);
  assert.ok(evidenceCase.indexOf("goLiveAcceptancePanel") < evidenceCase.indexOf("evidenceRunsPanel"));
  assert.match(i18n, /"section\.goLiveAcceptance": "上线验收"/);
  assert.match(i18n, /"text\.goLiveAcceptanceTaskTitle": "确认这次权限变更是否可以上线"/);
  assert.match(i18n, /"empty\.evidenceRuns\.detail": "历史自检运行会在这里保留；当前权限变更请以上方上线验收状态为准。"/);
  assert.match(styles, /\.go-live-acceptance\s*\{/);
});

test("workspace navigation is reflected in the URL hash", () => {
  assert.match(app, /const \[activeNav, setActiveNav\] = useState<NavKey>\(initialNavKey\)/);
  assert.match(app, /window\.history\.replaceState/);
  assert.match(app, /navHashFor\(activeView\.key\)/);
  assert.match(app, /window\.addEventListener\("hashchange"/);
  assert.match(app, /navKeyFromHash\(window\.location\.hash\)/);
});

test("permission request blocks main actions when sample fallback data is shown", () => {
  assert.match(workbench, /liveDataAvailable: boolean/);
  assert.match(workbench, /const liveDataBlocked = !liveDataAvailable/);
  assert.match(workbench, /className="approval-live-warning"/);
  assert.match(workbench, /message\.fallbackDataModeTitle/);
  assert.match(workbench, /message\.fallbackDataModeDetail/);
  assert.match(workbench, /const permissionRequestBusy =/);
  assert.match(workbench, /disabled=\{liveDataBlocked \|\| permissionRequestBusy/);
  assert.match(workbench, /disabled=\{Boolean\(application\) \|\| liveDataBlocked \|\| permissionRequestBusy \|\| !canApply\}/);
  assert.match(styles, /\.approval-live-warning\s*\{/);
});

test("permission request approval decisions show reviewer context before resolving", () => {
  assert.match(workbench, /useState/);
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
  assert.match(workbench, /pendingApprovalDecision\.comment\.trim\(\)/);
  assert.match(workbench, /message\.permissionApprovalRejectReasonRequired/);
  assert.match(workbench, /onRejectApprovalRequest\(pendingApprovalDecision\.requestId, comment\)/);
  assert.match(workbench, /action\.confirmRejectPermissionRequest/);
  assert.match(workbench, /text\.approvalRejectReasonHelp/);
  assert.match(app, /async function rejectAiAdminApprovalRequest\(requestId\?: string, comment\?: string\)/);
  assert.match(app, /const reviewerComment = comment\?\.trim\(\)/);
  assert.match(app, /comment: reviewerComment/);
});

test("permission request can withdraw a pending approval request", () => {
  assert.match(workbench, /onWithdrawApprovalRequest: \(comment\?: string\) => void/);
  assert.match(workbench, /type ApprovalDecisionAction = "approve" \| "reject" \| "withdraw"/);
  assert.match(workbench, /beginApprovalDecision\("withdraw"/);
  assert.match(workbench, /action\.withdrawPermissionRequest/);
  assert.match(workbench, /action\.confirmWithdrawPermissionRequest/);
  assert.match(workbench, /text\.approvalWithdrawHelp/);
  assert.match(workbench, /approvalRequest\.status !== "pending" && approvalRequest\.status !== "approved"/);
  assert.match(workbench, /permissionApprovalStatusLabel\(approvalRequest\.status, t\)/);
  assert.match(i18n, /"status\.approvalWithdrawn"/);
  assert.match(i18n, /"message\.permissionApprovalWithdrawn"/);
  assert.match(app, /async function withdrawAiAdminApprovalRequest\(comment\?: string\)/);
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
  assert.match(workbench, /approvalRequest\?\.status === "approved" \|\| Boolean\(application\) \|\| goLiveReady/);
  assert.match(workbench, /const runtimeValidationReady = Boolean\(approvalJourneyResult\) \|\| goLiveReady/);
  assert.match(workbench, /runtimeValidationReady \? t\("text\.runtimeValidationResultReady"\) : t\("text\.runtimeValidationResultPending"\)/);
  assert.match(workbench, /disabled=\{Boolean\(application\) \|\| liveDataBlocked \|\| permissionRequestBusy \|\| !canApply\}/);
  assert.match(workbench, /application \? t\("action\.permissionPackageApplied"\) : applying \? t\("action\.applyingPermissionPackage"\) : t\("action\.applyPermissionPackage"\)/);
  assert.match(workbench, /approvalDisplayStatus: PermissionPackageApprovalRequest\["status"\] \| null/);
  assert.match(workbench, /const showPendingApprovalActions = !application && !goLiveReady && approvalRequest\?\.status === "pending"/);
  assert.match(workbench, /permissionPolicyGateDetailKey\(draft\.policyGate\.canApplyDirectly, approvalDisplayStatus\)/);
  assert.match(workbench, /showPolicyGateReasons && draft\.policyGate\.reasons\.length > 0/);
  assert.doesNotMatch(workbench, /<Badge tone=\{draft\.policyGate\.canApplyDirectly \? "success" : approvalRequest \? approvalStatusTone : "warning"\}>/);
  assert.match(workbench, /className="approval-completion"/);
  assert.match(workbench, /text\.permissionChangeCompleteTitle/);
  assert.match(workbench, /text\.permissionChangeCompleteDetail/);
  assert.match(workbench, /productionReadiness\?\.generatedAt/);
  assert.match(workbench, /action\.exportProductionEvidence/);
  assert.match(workbench, /action\.openAccessProfile/);
  assert.match(workbench, /action\.startPermissionApproval/);
  assert.match(workbench, /onOpenAccessProfile/);
  assert.match(workbench, /onStartNewPermissionChange/);
  assert.match(workbench, /const quickSecondaryActionLabel = goLiveReady/);
  assert.match(workbench, /const runQuickSecondaryAction = goLiveReady \? onOpenAccessProfile : onRunApprovalJourney/);
  assert.match(app, /function openAiAdminAccessProfile\(\)/);
  assert.match(app, /setActiveNav\("access"\)/);
  assert.match(app, /function startNewAiAdminPermissionChange\(\)/);
  assert.match(i18n, /"text\.policyGateApprovedDetail": "当前权限变更已记录审批。"/);
  assert.match(i18n, /"action\.permissionPackageApplied": "已应用"/);
  assert.match(styles, /\.approval-completion\s*\{/);
  assert.match(styles, /\.approval-completion-actions\s*\{/);
});

test("permission request advanced messages suppress successful load noise", () => {
  assert.match(workbench, /function shouldShowAdvancedStatusMessage/);
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
  const workspaceLabelStart = workbench.indexOf('approval-readonly-field');
  const technicalDetailsStart = workbench.indexOf('<details className="approval-details">');
  assert.notEqual(workspaceLabelStart, -1);
  assert.notEqual(technicalDetailsStart, -1);
  assert.ok(workspaceLabelStart < technicalDetailsStart);
  assert.match(workbench.slice(workspaceLabelStart, technicalDetailsStart), /workspaceName/);
  assert.doesNotMatch(workbench.slice(workspaceLabelStart, technicalDetailsStart), /form\.workspaceId/);
});

test("permission reviewer queue uses business labels before technical identifiers", () => {
  assert.match(workbench, /function permissionApprovalRequestBusinessLabel/);
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
  assert.match(styles, /\.approval-review-row-main\s*\{/);
  assert.match(styles, /\.approval-review-row-meta\s*\{/);
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
  assert.match(workbench, /className="approval-field is-wide"/);
  assert.match(workbench, /className="approval-readonly-field is-wide"/);
  assert.match(workbench, /className="approval-field approval-subject-field is-wide"/);
  assert.match(workbench, /className="approval-select is-wide"/);
  assert.match(styles, /\.approval-form-grid \.approval-dropdown-trigger span\s*\{[^}]*white-space:\s*normal;/s);
  assert.match(styles, /\.approval-form-grid \.approval-dropdown-trigger span\s*\{[^}]*text-overflow:\s*clip;/s);
  assert.match(styles, /@media \(max-width: 1120px\)\s*\{[\s\S]*\.approval-flow-layout,\s*[\s\S]*\.approval-form-grid\s*\{[^}]*grid-template-columns:\s*1fr;/s);
});

test("permission request dropdowns use deduplicated business labels", () => {
  assert.match(workbench, /function uniquePermissionEntityOptions/);
  assert.match(workbench, /const tenantOptions = uniquePermissionEntityOptions/);
  assert.match(workbench, /const callerOptions = uniquePermissionEntityOptions/);
  assert.match(workbench, /const targetOptions = uniquePermissionEntityOptions/);
  assert.match(workbench, /tenantDropdownOptions/);
  assert.match(workbench, /callerDropdownOptions/);
  assert.match(workbench, /targetDropdownOptions/);
});

test("permission request primary path avoids native select menus", () => {
  const formStart = workbench.indexOf('<div className="approval-form-grid">');
  const formEnd = workbench.indexOf('<div className="approval-package-preview"', formStart);
  const primaryForm = workbench.slice(formStart, formEnd);
  assert.notEqual(formStart, -1);
  assert.notEqual(formEnd, -1);
  assert.match(dropdown, /export function ApprovalDropdown/);
  assert.match(dropdown, /label: string/);
  assert.match(dropdown, /aria-labelledby=\{labelId\}/);
  assert.match(dropdown, /aria-activedescendant=\{activeOptionId\}/);
  assert.match(primaryForm, /<ApprovalDropdown/);
  assert.doesNotMatch(primaryForm, /<select/);
});

test("permission request chooses access objects instead of raw subject selectors", () => {
  const formStart = workbench.indexOf('<div className="approval-form-grid">');
  const formEnd = workbench.indexOf('<div className="approval-package-preview"', formStart);
  const primaryForm = workbench.slice(formStart, formEnd);
  const advancedStart = workbench.indexOf('<details className="approval-details">');
  const processStart = workbench.indexOf('<aside className="approval-process-panel"', advancedStart);
  const advancedForm = workbench.slice(advancedStart, processStart);

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
  assert.match(app, /import \{ CapabilityGovernanceView, type CapabilityGrantForm \} from "\.\/components\/CapabilityGovernanceView"/);
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
  assert.match(capabilityGovernanceView, /<ApprovalDropdown/);
  assert.doesNotMatch(capabilityGovernanceView, /<select/);
  assert.match(capabilityGovernanceView, /selectedAccessSubject\.id === customAccessSubjectOption\.id/);
});
