import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import {
  createTranslator,
  translationKeys,
  normalizeLanguage,
  resolveInitialLanguage
} from "../src/i18n.ts";

const app = readFileSync(new URL("../src/ConsoleController.tsx", import.meta.url), "utf8");
const i18nSource = readFileSync(new URL("../src/i18n.ts", import.meta.url), "utf8");

test("normalizeLanguage supports English and Simplified Chinese", () => {
  assert.equal(normalizeLanguage("zh-CN"), "zh-CN");
  assert.equal(normalizeLanguage("zh-Hans-CN"), "zh-CN");
  assert.equal(normalizeLanguage("en-US"), "en");
  assert.equal(normalizeLanguage("fr-FR"), "en");
});

test("resolveInitialLanguage prefers stored language over browser language", () => {
  assert.equal(resolveInitialLanguage("en", ["zh-CN"]), "en");
  assert.equal(resolveInitialLanguage(undefined, ["zh-CN", "en-US"]), "zh-CN");
  assert.equal(resolveInitialLanguage(undefined, ["en-US", "zh-CN"]), "en");
});

test("Simplified Chinese product copy avoids forensic wording", () => {
  assert.doesNotMatch(i18nSource, /\u8bc1\u636e/);
});

test("English product copy uses acceptance and records wording", () => {
  const t = createTranslator("en");

  assert.equal(t("action.exportProductionEvidence"), "Export acceptance report");
  assert.equal(t("concept.evidence"), "Acceptance materials");
  assert.equal(t("metric.runtimeEvidence"), "Runtime Records");
  assert.equal(t("metric.traceEvidence"), "Trace Records");
  assert.equal(t("navGroup.audit"), "Audit & Acceptance");
  assert.equal(t("panel.evidenceRuns"), "Acceptance History");
  assert.equal(t("section.aiAdminApprovalJourney"), "Runtime Validation Records");
  assert.equal(t("text.cockpitKeyMessageEvidence"), "Clear go-live status");
  assert.equal(t("message.productionEvidenceExported"), "Acceptance report exported.");
  assert.equal(
    t("message.permissionApprovalAlreadyConsumedRecovery"),
    "This approval has already been used. Refresh status checks or review the current permission change before retrying."
  );
  assert.equal(
    t("message.apiContractUnavailable"),
    "API compatibility check is unavailable. Upgrade the AgentHarbor API before using this console."
  );
  assert.equal(
    t("message.apiContractIncompatible"),
    "API is missing capabilities required by this console: {capabilities}. Upgrade the AgentHarbor API before continuing."
  );
  assert.equal(t("connectionDiagnostics.action"), "Run diagnostics");
  assert.equal(t("connectionDiagnostics.title"), "Connection diagnostics");
  assert.equal(t("connectionDiagnostics.summaryOk"), "Ready for the production journey.");
  assert.equal(t("connectionDiagnostics.session.title"), "Console session");
  assert.equal(t("connectionDiagnostics.mcp.error"), "Tool service check failed: {detail}");
});

test("administrator boundary workspace has complete bilingual copy", () => {
  const keys = [
    "page.adminAccess",
    "nav.admin-access",
    "navDetail.adminAccess",
    "adminAccess.create",
    "adminAccess.title",
    "adminAccess.subtitle",
    "adminAccess.oneTimeKey",
    "adminAccess.oneTimeKeyDetail",
    "adminAccess.forbiddenDetail",
    "adminAccess.forbiddenTitle",
    "adminAccess.role.platform_admin",
    "adminAccess.role.tenant_admin",
    "adminAccess.role.security_reviewer",
    "adminAccess.source.bootstrap",
    "adminAccess.source.managed",
    "adminAccess.status.active",
    "adminAccess.status.disabled",
    "message.adminAccessCreated",
    "message.adminAccessCreatedReloadFailed",
    "message.adminAccessRotated",
    "message.adminAccessRotatedReloadFailed",
    "message.adminAccessDisabled",
    "message.adminAccessDisabledReloadFailed",
    "error.adminAccessOperation",
    "error.adminAccessPlatformRequired"
  ];

  for (const language of ["en", "zh-CN"]) {
    const t = createTranslator(language);
    for (const key of keys) {
      assert.notEqual(t(key), key, `${language} should define ${key}`);
    }
  }
});

test("tenant permission center copy is bilingual", () => {
  const keys = [
    "tenantCenter.adminBoundary",
    "tenantCenter.adminBoundaryDetail",
    "tenantCenter.capabilities",
    "tenantCenter.dataScopes",
    "tenantCenter.empty.noCapabilities",
    "tenantCenter.empty.noWorkspaces",
    "tenantCenter.manageAdmins",
    "tenantCenter.openAccessProfile",
    "tenantCenter.operatorBoundary",
    "tenantCenter.permissionPackages",
    "tenantCenter.snapshot",
    "tenantCenter.startPermissionChange",
    "tenantCenter.status.blocked",
    "tenantCenter.status.needs_review",
    "tenantCenter.status.ready",
    "tenantCenter.workspaces",
  ];
  for (const language of ["en", "zh-CN"]) {
    const t = createTranslator(language);
    for (const key of keys) {
      assert.notEqual(t(key), key, `${language} missing ${key}`);
    }
  }
});

test("resource action context copy is bilingual", () => {
  const keys = [
    "action.processing",
    "message.duplicateResourceMutation",
    "resource.advanced.action",
    "resource.advanced.detail",
    "resource.advanced.title",
    "resource.actionContext.resource",
    "resource.actionContext.scope",
    "resource.actionContext.title",
    "resource.detail.ready"
  ];

  for (const language of ["en", "zh-CN"]) {
    const t = createTranslator(language);
    for (const key of keys) {
      assert.notEqual(t(key), key, `${language} missing ${key}`);
    }
  }
  assert.equal(createTranslator("en")("resource.detail.ready"), "The resource is ready; keep reviewing runtime activity.");
});

test("createTranslator returns core journey Chinese labels", () => {
  const t = createTranslator("zh-CN");

  assert.equal(t("app.title"), "AgentHarbor 控制平面");
  assert.equal(t("section.connectionSettings"), "连接设置");
  assert.equal(t("text.connectionSettingsDetail"), "仅在调试或兼容旧部署时配置管理密钥覆盖，并查看当前数据源。");
  assert.equal(t("text.adminKeyStoredLocally"), "覆盖密钥只保存在当前浏览器会话；生产环境建议使用登录会话。");
  assert.equal(t("auth.title"), "登录 AgentHarbor");
  assert.equal(t("auth.adminKeyLabel"), "管理员密钥");
  assert.equal(t("auth.securityNote"), "密钥只用于换取 HttpOnly 会话 Cookie，不会保存在前端界面里。");
  assert.equal(t("control.adminKeyOverride"), "管理员密钥覆盖");
  assert.equal(t("action.signOut"), "退出登录");
  assert.equal(t("nav.cockpit"), "系统自检");
  assert.equal(t("nav.access"), "权限画像");
  assert.equal(t("nav.traces"), "运行审计");
  assert.equal(t("nav.evidence"), "上线检查");
  assert.equal(t("nav.admin-access"), "管理员与边界");
  assert.equal(t("nav.registry"), "资源管理");
  assert.equal(t("nav.tenants"), "租户与组织");
  assert.equal(t("navGroup.primary"), "查与改");
  assert.equal(t("navGroup.audit"), "审计与验收");
  assert.equal(t("navGroup.configuration"), "资源清单");
  assert.equal(t("navDetail.ai-admin"), "新建、审批、应用并验收权限变更。");
  assert.equal(t("navDetail.adminAccess"), "管理管理员登录密钥和租户边界。");
  assert.equal(t("navDetail.access"), "按租户和工作区盘点生效权限。");
  assert.equal(t("navDetail.tenants"), "管理租户边界，并从租户发起权限变更。");
  assert.equal(t("navDetail.traces"), "复核运行时允许和拒绝调用。");
  assert.equal(t("page.cockpit"), "系统自检");
  assert.equal(t("page.access"), "租户权限控制台");
  assert.equal(t("page.adminAccess"), "管理员与边界");
  assert.equal(t("page.evidence"), "上线检查");
  assert.equal(t("page.registry"), "资源管理");
  assert.equal(t("page.tenants"), "租户与组织");
  assert.equal(t("page.traces"), "运行审计");
  assert.equal(t("panel.auditTraces"), "运行审计");
  assert.equal(t("empty.auditTraces.title"), "暂无运行记录");
  assert.equal(t("empty.auditTraces.detail"), "允许、拒绝和工具发现过滤记录会显示在这里。");
  assert.equal(t("panel.evidenceRuns"), "历史验收");
  assert.equal(t("panel.resourceLifecycle"), "资源管理");
  assert.equal(t("resource.status.needsApproval"), "需要授权");
  assert.equal(t("resource.detail.needsCredentials"), "需要先配置目标服务凭据。");
  assert.equal(t("resource.detail.needsCapabilities"), "需要发现并复核可用能力。");
  assert.equal(t("resource.detail.needsApproval"), "需要创建权限变更并完成授权。");
  assert.equal(t("resource.detail.needsRuntime"), "需要一次真实调用来确认运行结果。");
  assert.equal(t("resource.permissionIntent"), "为 {target} 创建授权。");
  assert.equal(t("resource.contextTitle"), "当前资源");
  assert.equal(t("resource.contextDetail"), "所选资源会决定推荐操作，同时保留租户和工作区上下文。");
  assert.equal(t("resource.advanced.title"), "高级资源明细");
  assert.equal(t("resource.advanced.action"), "查看明细");
  assert.equal(t("resource.advanced.detail"), "仅在排查问题、批量核对或审计复核时展开 Agent 注册表和契约矩阵。");
  assert.equal(t("resource.contextScope"), "范围");
  assert.equal(t("resource.contextHealth"), "状态");
  assert.equal(t("resource.contextNext"), "推荐下一步");
  assert.equal(t("resource.listTitle"), "资源清单");
  assert.equal(t("resource.listCount").replace("{count}", "3"), "3 个资源");
  assert.equal(t("resource.actionContext.title"), "操作上下文");
  assert.equal(t("resource.actionContext.resource"), "资源");
  assert.equal(t("resource.actionContext.scope"), "操作范围");
  assert.equal(t("text.permissionHandoffRegistryTitle"), "已从资源管理带入");
  assert.equal(t("resource.nextAction.capabilities"), "发现能力");
  assert.equal(t("empty.evidenceRuns.title"), "暂无历史验收记录");
  assert.equal(t("section.goLiveAcceptance"), "上线检查");
  assert.equal(t("productionAcceptance.title"), "上线检查");
  assert.equal(t("productionAcceptance.status.ready"), "可上线");
  assert.equal(t("productionAcceptance.blockers"), "阻断项");
  assert.equal(t("productionAcceptance.blocker.liveData"), "实时 API 数据尚未连接。");
  assert.equal(t("text.goLiveAcceptanceTaskTitle"), "确认这次权限变更是否可以上线");
  assert.equal(t("text.goLiveAcceptanceNoReadinessDetail"), "先回到权限变更工作台执行运行验证或状态检查，系统会补齐运行、权限画像和审计记录。");
  assert.equal(t("metric.runtimeEvidence"), "运行记录");
  assert.equal(t("action.loadProfile"), "加载画像");
  assert.equal(t("panel.coreJourney"), "核心权限链路自检");
  assert.equal(t("action.runCoreJourney"), "执行自检");
  assert.equal(t("action.resetCoreJourney"), "重置会话");
  assert.equal(t("section.systemHealthStatus"), "系统健康状态");
  assert.equal(t("section.selfCheckTask"), "检查任务");
  assert.equal(t("section.selfCheckAdvanced"), "高级配置");
  assert.equal(t("section.selfCheckRuntimeDetail"), "运行明细");
  assert.equal(t("status.systemHealthReady"), "可执行");
  assert.equal(t("status.systemHealthNeedsCheck"), "需检查");
  assert.equal(t("text.cockpitKeyMessageTitle"), "让 Agent 的工具和数据访问可申请、可审批、可验证");
  assert.equal(t("text.cockpitKeyMessageBody"), "AgentHarbor 以租户为边界，把权限包、审批、运行拦截和上线检查串成一条可交付的生产路径。");
  assert.equal(t("text.cockpitKeyMessageBoundary"), "租户边界清楚");
  assert.equal(t("text.cockpitKeyMessageBoundaryDetail"), "三级租户、工作区和实例权限共同决定可访问数据。");
  assert.equal(t("text.cockpitKeyMessageOperations"), "权限变更可控");
  assert.equal(t("text.cockpitKeyMessageOperationsDetail"), "管理员新建权限变更，高风险能力先审批再落地。");
  assert.equal(t("text.cockpitKeyMessageEvidence"), "上线状态清楚");
  assert.equal(t("text.cockpitKeyMessageEvidenceDetail"), "允许/拒绝调用、访问画像和审计事件共同支撑上线判断。");
  assert.equal(t("text.coreJourneyIntro"), "验证租户、工具发现、授权链、运行拦截和访问画像是否形成最小权限闭环。");
  assert.equal(t("text.coreJourneyCompletion"), "自检进度");
  assert.equal(t("text.systemHealthReadyTitle"), "可以执行自检");
  assert.equal(t("text.systemHealthNeedsCheckTitle"), "需要先完成预检");
  assert.equal(t("text.selfCheckAdvancedDetail"), "仅在切换工作区、端点或工具名称时调整这些配置。");
  assert.equal(t("preflight.api.title"), "API 服务");
  assert.equal(t("preflight.mockMcp.title"), "MCP 工具服务");
  assert.equal(t("journey.step.grantChain"), "租户/工作区/实例授权");
  assert.equal(t("coreJourney.stepDetail.tenantTree.missing"), "等待创建三级租户范围。");
  assert.equal(t("coreJourney.stepDetail.agentPair.complete"), "调用方和目标服务已匹配。");
  assert.equal(t("coreJourney.stepDetail.runtimeEvidence.complete"), "允许和拒绝调用均已验证。");
  assert.equal(t("traceRoute.mcpToolsCall"), "工具调用");
  assert.equal(t("traceRoute.mcpToolsList"), "工具发现");
  assert.equal(t("traceReason.filteredToolsListByCapabilityAssignments"), "工具列表已按权限收敛");
  assert.equal(t("traceReason.capabilityNotApproved"), "能力未审批，已拒绝");
});

test("createTranslator returns Chinese labels for operator controls", () => {
  const t = createTranslator("zh-CN");

  assert.equal(t("form.name"), "名称");
  assert.equal(t("form.channel"), "通道");
  assert.equal(t("form.credentialHeader"), "凭据 Header");
  assert.equal(t("action.createAgent"), "创建 Agent");
  assert.equal(t("action.rotateCredential"), "轮换凭据");
  assert.equal(t("action.openResourceManagement"), "打开资源管理");
  assert.equal(t("table.policy"), "策略");
  assert.equal(t("form.decision"), "判定结果");
  assert.equal(t("form.traceRunId"), "运行批次");
  assert.equal(t("form.traceRunPlaceholder"), "可选运行批次");
  assert.equal(t("status.agentDraft"), "草稿");
  assert.equal(t("status.policyAllow"), "允许");
  assert.equal(t("empty.routePolicies.title"), "暂无路由策略");
  assert.equal(t("empty.routePolicies.detail"), "请在资源管理中创建策略；完成资源配置后，受治理的路由会显示在这里。");
  assert.equal(t("resource.detail.ready"), "资源已就绪，可继续查看运行情况。");
  assert.equal(t("resource.refreshStatus.staleDetail"), "继续处理该资源变更前请先刷新。");
  assert.equal(t("resource.stagesAria"), "资源配置步骤");
  assert.equal(t("auditAction.permission_package.applied"), "应用权限包");
  assert.equal(t("auditActor.local-dev"), "本地开发管理员");
});

test("createTranslator returns Chinese labels for AI admin permission packages", () => {
  const t = createTranslator("zh-CN");

  assert.equal(t("nav.ai-admin"), "权限变更");
  assert.notEqual(t("nav.ai-admin"), "申请权限");
  assert.equal(t("page.aiAdmin"), "权限变更与状态检查");
  assert.equal(t("panel.aiAdminPermissionWorkbench"), "权限变更工作台");
  assert.equal(t("section.aiAdminApprovalJourney"), "运行验证记录");
  assert.equal(t("section.aiAdminReadiness"), "环境检查");
  assert.equal(t("section.aiAdminProductionConsole"), "状态检查");
  assert.equal(t("action.runApprovalJourney"), "执行运行验证");
  assert.equal(t("action.startPermissionApproval"), "新建权限变更");
  assert.equal(t("action.startTenantPermissionChange"), "新建权限变更");
  assert.equal(t("action.runningApprovalJourney"), "验证中");
  assert.equal(t("action.checkApprovalReadiness"), "检查环境");
  assert.equal(t("text.aiAdminApprovalJourneyCompletion"), "状态检查进度");
  assert.equal(t("text.aiAdminGoLiveReadyTitle"), "已满足上线条件");
  assert.equal(t("text.aiAdminGoLiveRemainingBadge"), "还差 {count}");
  assert.equal(t("text.aiAdminGoLiveWaitingTitle"), "暂不能上线，还差 {count} 步");
  assert.equal(t("text.aiAdminGoLiveWaitingDetail"), "按下一步补齐审批、应用、运行验证和审计记录。");
  assert.equal(t("journey.aiAdmin.next.approvalRequest"), "先发起并批准审批请求。");
  assert.equal(t("text.aiAdminProductionConsoleHelp"), "按申请、审批、落地和运行验证汇总状态检查结果。");
  assert.equal(t("section.permissionRequestTask"), "权限变更");
  assert.equal(t("section.permissionWizardScope"), "选择对象");
  assert.equal(t("section.permissionWizardTemplate"), "选择权限");
  assert.equal(t("section.permissionWizardApproval"), "审批处理");
  assert.equal(t("permissionWorkbench.step.request"), "配置范围");
  assert.equal(t("permissionWorkbench.step.approval"), "审批处理");
  assert.equal(t("text.permissionProcessStepAria"), "第 {index} 步：{label}，{detail}");
  assert.equal(t("section.permissionWizardApply"), "应用权限");
  assert.equal(t("section.permissionWizardGoLive"), "状态检查");
  assert.equal(t("demo.coreJourneyTarget"), "标准工具服务");
  assert.equal(t("demo.coreJourneyWorkspace"), "平台工作区");
  assert.equal(t("demo.mcpCapabilityCaller"), "客服助手");
  assert.equal(t("demo.mcpCapabilityTarget"), "工单工具服务");
  assert.doesNotMatch(t("demo.coreJourneyCaller"), /自检/);
  assert.doesNotMatch(t("demo.coreJourneyProject"), /自检/);
  assert.doesNotMatch(t("demo.coreJourneyRoot"), /自检/);
  assert.doesNotMatch(t("demo.coreJourneyTarget"), /自检/);
  assert.doesNotMatch(t("demo.coreJourneyTeam"), /自检/);
  assert.equal(t("demo.permissionRequestApprovalTeam"), "客户服务中心");
  assert.equal(t("demo.permissionRequestApprovalTarget"), "工单工具服务");
  assert.equal(t("demo.permissionRequestWorkspace"), "客户服务工作区");
  assert.doesNotMatch(t("demo.permissionRequestApprovalRoot"), /权限申请/);
  assert.doesNotMatch(t("demo.permissionRequestApprovalTeam"), /权限申请/);
  assert.doesNotMatch(t("demo.permissionRequestApprovalProject"), /权限申请/);
  assert.doesNotMatch(t("demo.permissionRequestApprovalCaller"), /权限申请/);
  assert.doesNotMatch(t("demo.permissionRequestApprovalTarget"), /权限申请/);
  assert.equal(t("section.permissionAdvancedChecks"), "验收明细");
  assert.equal(t("form.businessTenant"), "租户");
  assert.equal(t("form.businessWorkspace"), "工作区");
  assert.equal(t("form.businessCaller"), "调用方");
  assert.equal(t("form.accessSubject"), "访问对象");
  assert.equal(t("accessSubject.kind.role"), "角色");
  assert.equal(t("accessSubject.supportAgent.name"), "客服专员");
  assert.equal(t("accessSubject.supportAgent.detail"), "推荐选择。适用于当前工作区内处理客户工单的一线客服。");
  assert.equal(t("accessSubject.securityReviewer.name"), "安全审批人");
  assert.equal(t("dataScope.us-east"), "美东");
  assert.equal(t("text.tenantLevel.0"), "1级租户");
  assert.equal(t("tenantOrg.permissionManagement"), "租户权限在「权限变更」里管理");
  assert.equal(t("tenantOrg.permissionManagementDetail"), "从这里发起变更，审批、应用、状态检查和审计记录都会保留。");
  assert.equal(t("tenantOrg.permissionIntent"), "通过受控的权限变更流程为该租户创建或调整权限。");
  assert.equal(t("tenantOrg.startPermissionModalTitle"), "新建权限变更");
  assert.equal(t("tenantOrg.startPermissionSubmit"), "进入权限变更");
  assert.equal(t("tenantOrg.accessDirectory"), "访问对象目录");
  assert.equal(t("accessSubject.support002.name"), "张敏");
  assert.equal(t("accessSubject.supportLead001.name"), "李航");
  assert.equal(t("text.permissionTenantHandoffTitle"), "已从租户与组织带入");
  assert.equal(t("text.permissionTenantHandoffDetail"), "{tenant} / {workspace}；在这里创建这个租户范围内的权限变更。");
  assert.equal(t("text.technicalDetails"), "技术详情");
  assert.equal(t("text.technicalOverrides"), "技术覆盖");
  assert.equal(t("text.filterSettings"), "筛选条件");
  assert.equal(t("text.traceDetails"), "追踪详情");
  assert.equal(t("action.expandTraceDetails"), "展开追踪详情");
  assert.equal(t("action.collapseTraceDetails"), "收起追踪详情");
  assert.equal(t("text.tenantPath"), "租户层级");
  assert.equal(t("text.workspaceAlias"), "工作区名称");
  assert.equal(t("text.defaultTenantName"), "集团总部");
  assert.equal(t("text.defaultWorkspaceName"), "客户服务工作区");
  assert.equal(t("text.unresolvedTenant"), "未选择租户");
  assert.equal(t("text.permissionRequestTaskTitle"), "权限变更配置");
  assert.equal(t("text.permissionRequestTaskBody"), "配置租户、调用方和权限包；审批通过后应用变更，并完成状态检查。");
  assert.equal(t("text.permissionRequestStatusSummary"), "权限变更状态摘要");
  assert.equal(t("text.currentWorkspaceContext"), "当前工作上下文");
  assert.equal(t("text.permissionRequestNextAction"), "建议下一步");
  assert.equal(t("text.permissionRequestScopeHelp"), "选择业务租户、调用方、工具服务和访问角色；技术 ID 只在技术覆盖里保留。");
  assert.equal(t("section.permissionRequestReview"), "配置复核");
  assert.equal(t("text.permissionRequestReviewHelp"), "复核本次生效的租户、调用方、工具服务、访问角色和权限包；需要调整时请新建变更。");
  assert.equal(t("text.permissionRequestTemplateHelp"), "从模板开始，系统会自动带出允许工具、阻断工具和数据范围。");
  assert.equal(t("text.permissionRequestApprovalHelp"), "系统会判断是否需要审批；审批通过后再应用到生产授权链。");
  assert.equal(t("text.permissionRequestApplyHelp"), "应用已批准的权限包，并确认权限已经生效。");
  assert.equal(t("text.permissionApplyReadyTitle"), "可以应用权限");
  assert.equal(t("text.permissionApplyWaitingTitle"), "等待审批");
  assert.equal(t("status.currentStep"), "当前");
  assert.equal(t("text.permissionRequestGoLiveHelp"), "应用后执行运行验证，系统验证允许/拒绝调用并生成交接材料。");
  assert.equal(t("text.permissionRequestAdvancedSummary"), "主流程背后的辅助材料：环境、预检、审批队列、状态检查、权限判定、影响和运行记录。");
  assert.equal(t("text.permissionRequestLockedTitle"), "当前配置仅供复核");
  assert.equal(t("text.permissionRequestLockedActiveDetail"), "本次权限变更已经生效。如需修改范围或权限包，请新建权限变更。");
  assert.equal(t("text.permissionRequestLockedApprovalDetail"), "申请已进入审批冻结状态。如需修改范围或权限包，请撤回后重新发起。");
  assert.equal(t("text.workspaceResolvedDetail"), "由所选租户和调用方确定；只有高级操作时才需要手动覆盖。");
  assert.equal(t("action.runDemoPermissionRequest"), "执行运行验证");
  assert.equal(t("text.permissionAppliedTitle"), "权限已生效");
  assert.equal(t("text.productionReadinessReadyTitle"), "可上线");
  assert.equal(t("text.applicationHealthReadyDetail"), "权限已生效，未发现漂移。");
  assert.equal(t("text.approvalExpiresAt"), "{date} 到期");
  assert.equal(t("text.runtimeValidationResultTitle"), "运行验证结果");
  assert.equal(t("text.runtimeValidationResultReady"), "授权访问已放行，阻断访问已拒绝。");
  assert.equal(t("productionConsole.request"), "变更申请");
  assert.equal(t("productionConsole.policyGate"), "策略门禁");
  assert.equal(t("productionConsole.approval"), "审批");
  assert.equal(t("productionConsole.application"), "权限落地");
  assert.equal(t("productionConsole.runtime"), "运行验证");
  assert.equal(t("productionConsole.productionReadiness"), "状态检查");
  assert.equal(t("productionConsole.approvalSatisfied"), "审批已满足");
  assert.equal(t("productionConsole.requestConfigured"), "申请已配置");
  assert.equal(t("productionConsole.requestNeedsInput"), "申请待补充");
  assert.equal(t("productionConsole.approvalNotRequired"), "无需审批");
  assert.equal(t("productionConsole.applicationPending"), "待应用权限");
  assert.equal(t("productionConsole.runtimeEvidence"), "等待运行验证");
  assert.equal(t("productionConsole.runtimeReady"), "运行验证已通过");
  assert.equal(t("productionConsole.productionReady"), "已满足上线条件");
  assert.equal(t("readiness.aiAdmin.api.title"), "控制台连接");
  assert.equal(t("message.apiContractUnavailable"), "无法读取 API 兼容信息。请先升级 AgentHarbor API，再继续使用当前控制台。");
  assert.equal(t("message.apiContractIncompatible"), "API 缺少当前控制台需要的能力：{capabilities}。请先升级 AgentHarbor API。");
  assert.equal(t("connectionDiagnostics.action"), "运行诊断");
  assert.equal(t("connectionDiagnostics.title"), "连接诊断");
  assert.equal(t("connectionDiagnostics.summaryOk"), "可以继续生产主流程。");
  assert.equal(t("connectionDiagnostics.session.title"), "控制台会话");
  assert.equal(t("connectionDiagnostics.mcp.error"), "工具服务检查失败：{detail}");
  assert.equal(t("readiness.aiAdmin.mockMcp.title"), "工具服务连接");
  assert.equal(t("readiness.aiAdmin.subjectHeader.title"), "浏览器身份可用");
  assert.equal(t("readiness.aiAdmin.privateUpstreams.title"), "本地演示连接");
  assert.equal(t("readiness.aiAdmin.privateUpstreams.detail"), "本地演示环境已允许连接工具服务。");
  assert.equal(t("journey.aiAdmin.step.approvedApply"), "权限包落地");
  assert.equal(t("journey.aiAdmin.evidence.approvedApply"), "确认已批准的权限变更已经落地并生成应用记录。");
  assert.equal(t("section.permissionApplicationEvidence"), "权限已落地");
  assert.equal(t("detail.applicationId"), "应用记录");
  assert.equal(t("section.permissionDraft"), "申请配置");
  assert.equal(t("section.permissionPolicyGate"), "策略门禁");
  assert.equal(t("section.permissionSimulation"), "发布前模拟");
  assert.equal(t("form.permissionPackage"), "权限包模板");
  assert.equal(t("action.applyPermissionPackage"), "应用权限");
  assert.equal(t("action.permissionPackageApplied"), "已应用");
  assert.equal(t("action.approvePermissionRequest"), "批准请求");
  assert.equal(t("action.confirmApprovePermissionRequest"), "确认批准");
  assert.equal(t("action.confirmRejectPermissionRequest"), "确认拒绝");
  assert.equal(t("action.cancelApprovalDecision"), "取消");
  assert.equal(t("action.createApprovalRequest"), "提交审批");
  assert.equal(t("empty.permissionApplyPreflight.detail"), "应用权限前先运行预检。");
  assert.equal(t("message.permissionPackageApprovalRequired"), "该申请需要先审批：{detail}。");
  assert.equal(t("message.permissionPackageReadinessWarning"), "应用前请先复核这次权限变更。");
  assert.equal(t("section.permissionApprovalRequest"), "审批请求");
  assert.equal(t("section.permissionApprovalTrace"), "审批追溯");
  assert.equal(t("section.permissionReviewerQueue"), "待审批请求");
  assert.equal(t("form.approvalReviewer"), "审批人");
  assert.equal(t("form.approvalRejectReason"), "拒绝理由");
  assert.equal(t("text.approvalReviewerIdentity"), "当前审批人：{reviewer}");
  assert.equal(t("text.approvalReviewerSeparationDetail"), "生产环境会校验具名管理员身份，并强制职责分离。");
  assert.equal(t("text.approvalRejectReasonHelp"), "用于审计和交接，不能为空。");
  assert.equal(t("action.refreshApprovalTrace"), "刷新追溯");
  assert.equal(t("action.refreshReviewerQueue"), "刷新队列");
  assert.equal(t("empty.reviewerQueue.detail"), "输入审批人标识后刷新待处理请求。");
  assert.equal(t("message.reviewerQueueLoaded"), "已加载 {count} 个待审批请求。");
  assert.equal(t("section.accessDecisionExplain"), "权限判定说明");
  assert.equal(t("section.accessProfileTask"), "租户访问画像");
  assert.equal(t("section.accessProfileFilters"), "查询范围");
  assert.equal(t("section.accessProfileAdjustScope"), "调整查看范围");
  assert.equal(t("text.accessProfileTaskTitle"), "复核最终访问权限");
  assert.equal(t("text.accessProfileFiltersDetail"), "先选择租户、工作区、调用方、目标和能力，再查看最终权限。");
  assert.equal(t("text.accessProfileAdjustScopeDetail"), "本次权限变更范围已固定在上方；只有需要查看其他验收范围时再调整筛选。");
  assert.equal(t("text.accessProfileHandoffContext"), "权限变更交接上下文");
  assert.equal(t("text.accessProfileHandoffTitle"), "正在复核同一次权限变更");
  assert.equal(t("text.accessProfileHandoffDetail"), "同一次权限变更范围已固定在下方，包含租户、工作区、调用方、目标和能力。");
  assert.equal(t("action.explainAccessDecision"), "查看判定");
  assert.equal(t("form.subjectId"), "主体 ID");
  assert.equal(t("text.notSpecified"), "未指定");
  assert.equal(t("empty.accessDecisionExplain.detail"), "选择租户、工作区、调用方、目标和能力后查看判定原因。");
  assert.equal(t("message.accessDecisionExplainLoaded"), "权限判定已加载。");
  assert.equal(t("message.accessDecisionExplainMissingFields"), "请先选择租户、工作区、调用方、目标和能力，再查看权限判定。");
  assert.equal(t("message.validationRetryAttempts"), "重试次数必须是 1 到 4 之间的整数。");
  assert.equal(t("message.validationRetryBackoff"), "重试退避必须是 0 到 1000 毫秒之间的整数。");
  assert.equal(t("text.nextActions"), "下一步动作");
  assert.equal(t("section.permissionApplicationImpact"), "影响范围");
  assert.equal(t("section.permissionApplicationHealth"), "落地结果");
  assert.equal(t("section.permissionProductionReadiness"), "状态检查");
  assert.equal(t("action.reviewApplicationImpact"), "查看影响");
  assert.equal(t("action.refreshApplicationHealth"), "刷新状态");
  assert.equal(t("action.checkProductionReadiness"), "执行状态检查");
  assert.equal(t("action.exportProductionEvidence"), "导出验收报告");
  assert.equal(t("action.openAcceptanceDetails"), "查看验收明细");
  assert.equal(t("action.openProcessDetails"), "查看处理流程");
  assert.equal(t("action.openAccessProfile"), "查看权限画像");
  assert.equal(t("action.withdrawPermissionRequest"), "撤回请求");
  assert.equal(t("action.confirmWithdrawPermissionRequest"), "确认撤回");
  assert.equal(t("action.rehearseApplicationDrift"), "演练漂移");
  assert.equal(t("form.requiredFieldFallback"), "必填项");
  assert.equal(t("text.permissionChangeCompleteTitle"), "本次权限变更已生效");
  assert.equal(t("text.permissionChangeCompleteDetail"), "权限已应用，运行验证和上线检查已完成。");
  assert.equal(t("text.permissionChangeCompletedAt"), "完成时间：{date}");
  assert.equal(t("message.permissionApplicationImpactLoaded"), "影响复核已加载。");
  assert.equal(t("message.permissionApplicationHealthLoaded"), "落地状态已加载。");
  assert.equal(t("message.permissionProductionReadinessLoaded"), "状态检查结果已加载。");
  assert.equal(t("message.productionEvidenceExported"), "上线状态报告已导出。");
  assert.equal(t("message.productionEvidenceRequiresLiveApi"), "导出上线状态报告需要实时 API。");
  assert.equal(t("message.permissionProductionReadinessRequiresLiveApi"), "状态检查需要实时 API。");
  assert.equal(t("message.permissionApplicationHealthRequiresLiveApi"), "落地状态巡检需要实时 API。");
  assert.equal(t("message.permissionApplicationDriftRehearsalLoaded"), "漂移演练已加载。");
  assert.equal(t("message.permissionApplicationImpactRequiresLiveApi"), "影响复核需要实时 API。");
  assert.equal(t("metric.productionReadyChecks"), "通过检查");
  assert.equal(t("metric.productionBlockers"), "上线阻断");
  assert.equal(t("metric.productionWarnings"), "上线提示");
  assert.equal(t("metric.activeObjects"), "有效对象");
  assert.equal(t("metric.missingObjects"), "缺失对象");
  assert.equal(t("metric.readyApplications"), "正常应用");
  assert.equal(t("metric.driftedApplications"), "漂移应用");
  assert.equal(t("metric.needsReviewApplications"), "需复核应用");
  assert.equal(t("metric.totalApplications"), "应用总数");
  assert.equal(t("text.rollbackReview"), "回滚评审");
  assert.equal(t("rollbackStep.capabilityManualReview"), "人工复核能力发现状态；共享能力不会被自动降级。");
  assert.equal(t("blocker.missing_created_objects"), "部分已记录授权对象缺失；回滚前需要先调查漂移。");
  assert.equal(t("blocker.inactive_created_objects"), "部分已记录授权对象未启用；需要复核是否已经发生手工变更或部分处置。");
  assert.equal(t("blocker.no_allowed_capabilities"), "该应用没有记录允许能力；不能按自动顺序规划回滚。");
  assert.equal(t("blocker.unknown"), "该影响项需要人工复核后再确认上线。");
  assert.equal(t("text.remediationPlan"), "处置计划");
  assert.equal(t("text.readOnlyPlan"), "只读计划");
  assert.equal(t("text.rehearsalMode"), "只读演练");
  assert.equal(t("rehearsal.grant_drift"), "授权漂移演练");
  assert.equal(t("text.rehearsalReadOnlyDetail"), "该结果仅模拟缺失或未启用的授权对象，不会写入权限变更。");
  assert.equal(t("metric.plannedActions"), "计划动作");
  assert.equal(t("text.finalAccessVerification"), "最终访问校验");
  assert.equal(t("text.capabilityReview"), "能力评审");
  assert.equal(t("text.workspaceAssignment"), "工作区分配");
  assert.equal(t("text.instanceAssignment"), "实例分配");
  assert.equal(t("empty.permissionApplicationImpact.detail"), "先查看已应用权限的影响，再规划任何回滚动作。");
  assert.equal(t("empty.permissionApplicationHealth.detail"), "应用权限后刷新落地状态。");
  assert.equal(t("empty.remediationActions.detail"), "暂无处置动作。");
  assert.equal(t("value.read_only"), "只读");
  assert.equal(t("value.verify"), "校验");
  assert.equal(t("value.shared_capability_manual_review"), "共享能力需要人工评审，不能自动降级。");
  assert.equal(t("value.disable_instance_assignment"), "先禁用记录的实例分配，缩小调用方访问面。");
  assert.equal(t("value.verify_effective_access"), "手动处置后重新解释有效权限，确认访问已按预期收敛。");
  assert.equal(t("value.manual_review"), "人工评审");
  assert.equal(t("value.investigate"), "调查");
  assert.equal(t("permissionPolicy.unknownReason"), "这次权限变更需要先完成策略复核。");
  assert.equal(t("permissionPreflight.unknown"), "预检项");
  assert.equal(t("permissionPreflight.detail.unknown"), "应用前请先复核这项预检结果。");
  assert.equal(t("permissionPreflight.next.unknown"), "请复核预检结果后选择下一步安全动作。");
  assert.equal(t("status.directApplyAllowed"), "可直接应用");
  assert.equal(t("status.approvalRequired"), "需要审批");
  assert.equal(t("status.applicationHealthReady"), "健康");
  assert.equal(t("status.applicationHealthDrifted"), "已漂移");
  assert.equal(t("status.applicationHealthNeedsReview"), "需复核");
  assert.equal(t("status.productionReady"), "可上线");
  assert.equal(t("status.productionNeedsReview"), "需复核");
  assert.equal(t("status.productionBlocked"), "阻断");
  assert.equal(t("default.aiAdminApprovalJourneyRequestText"), "给客服分诊助手开通当前租户的客户查询和工单有限更新权限，禁止导出合同。");
  assert.equal(t("productionCheck.runtime_allowed_trace_present"), "允许运行记录");
  assert.equal(t("productionCheck.runtime_denied_trace_present"), "拒绝运行记录");
  assert.equal(t("productionCheck.unknown"), "状态检查项");
  assert.equal(t("productionCheck.detail.unknown"), "确认上线前请先复核这项状态检查。");
  assert.equal(t("empty.permissionProductionReadiness.detail"), "应用权限并采集运行、审计和落地记录后执行状态检查。");
  assert.equal(t("status.approvalPending"), "待审批");
  assert.equal(t("status.stepMissing"), "待完成");
  assert.equal(t("message.permissionApprovalAlreadyPending"), "当前申请已有待审批请求，请先查看或批准现有请求后再继续。");
  assert.equal(t("message.permissionApprovalPending"), "审批请求仍在待审，请先批准再应用。");
  assert.equal(t("message.permissionApprovalRejectReasonRequired"), "请先填写拒绝理由，再拒绝请求。");
  assert.equal(t("message.permissionApprovalRequestTextRequired"), "请先填写管理员需求，再提交审批。");
  assert.equal(t("message.permissionApprovalWithdrawn"), "审批请求已撤回，可以修改后重新提交。");
  assert.equal(
    t("message.permissionApprovalAlreadyConsumedRecovery"),
    "审批已被使用。请先刷新状态检查或查看当前权限变更，确认是否已应用；不要重复提交。"
  );
  assert.equal(t("status.approvalWithdrawn"), "已撤回");
  assert.equal(t("permissionSimulation.guardrailFinance"), "销售只读权限包不包含财务字段访问。");
  assert.equal(t("permissionPolicy.riskApprovalRequired"), "{capability} 是 {risk} 风险能力，需要先审批。");
  assert.equal(t("message.permissionPackageApprovalRequired"), "该申请需要先审批：{detail}。");
  assert.equal(t("message.noMatchingAllowedCapabilities"), "当前目标没有匹配的允许能力。");
});

test("createTranslator falls back to English for missing keys", () => {
  const t = createTranslator("zh-CN");

  assert.equal(t("missing.key", "Fallback"), "Fallback");
});

test("English and Simplified Chinese translation maps expose the same keys", () => {
  assert.deepEqual(translationKeys("zh-CN"), translationKeys("en"));
});

test("UI error fallbacks are localized instead of hard-coded English strings", () => {
  assert.equal(app.includes('"Unable to'), false);
  assert.equal(app.includes('"Core journey failed"'), false);
  assert.equal(app.includes('"Permission package approval journey failed"'), false);
  assert.equal(app.includes('"console data unavailable"'), false);
});
