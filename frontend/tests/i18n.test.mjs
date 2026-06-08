import assert from "node:assert/strict";
import test from "node:test";

import {
  createTranslator,
  normalizeLanguage,
  resolveInitialLanguage
} from "../src/i18n.ts";

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

test("createTranslator returns core journey Chinese labels", () => {
  const t = createTranslator("zh-CN");

  assert.equal(t("app.title"), "AgentHarbor 控制平面");
  assert.equal(t("control.search"), "搜索权限包、租户、调用方");
  assert.equal(t("nav.cockpit"), "自检");
  assert.equal(t("nav.access"), "权限");
  assert.equal(t("page.cockpit"), "系统自检");
  assert.equal(t("page.access"), "租户权限控制台");
  assert.equal(t("metric.runtimeEvidence"), "运行证据");
  assert.equal(t("action.loadProfile"), "加载画像");
  assert.equal(t("panel.coreJourney"), "核心权限链路自检");
  assert.equal(t("action.runCoreJourney"), "执行自检");
  assert.equal(t("action.resetCoreJourney"), "重置会话");
  assert.equal(t("text.cockpitKeyMessageTitle"), "让 Agent 的工具和数据访问可申请、可审批、可验证");
  assert.equal(t("text.cockpitKeyMessageBody"), "AgentHarbor 以租户为边界，把权限包、审批、运行拦截和证据验收串成一条可交付的生产路径。");
  assert.equal(t("text.cockpitKeyMessageBoundary"), "租户边界清楚");
  assert.equal(t("text.cockpitKeyMessageBoundaryDetail"), "三级租户、工作区和实例权限共同决定可访问数据。");
  assert.equal(t("text.cockpitKeyMessageOperations"), "权限变更可控");
  assert.equal(t("text.cockpitKeyMessageOperationsDetail"), "管理员通过权限包发起申请，高风险能力先审批再落地。");
  assert.equal(t("text.cockpitKeyMessageEvidence"), "上线证据完整");
  assert.equal(t("text.cockpitKeyMessageEvidenceDetail"), "允许/拒绝调用、访问画像和审计事件共同支撑验收。");
  assert.equal(t("text.coreJourneyIntro"), "验证租户、工具发现、授权链、运行拦截和访问画像是否形成最小权限闭环。");
  assert.equal(t("text.coreJourneyCompletion"), "自检进度");
  assert.equal(t("preflight.api.title"), "API 服务");
  assert.equal(t("preflight.mockMcp.title"), "Mock MCP 服务");
  assert.equal(t("journey.step.grantChain"), "租户/工作区/实例授权");
});

test("createTranslator returns Chinese labels for operator controls", () => {
  const t = createTranslator("zh-CN");

  assert.equal(t("form.name"), "名称");
  assert.equal(t("form.channel"), "通道");
  assert.equal(t("form.credentialHeader"), "凭据 Header");
  assert.equal(t("action.createAgent"), "创建 Agent");
  assert.equal(t("action.rotateCredential"), "轮换凭据");
  assert.equal(t("table.policy"), "策略");
  assert.equal(t("status.agentDraft"), "草稿");
  assert.equal(t("status.policyAllow"), "允许");
  assert.equal(t("empty.routePolicies.title"), "暂无路由策略");
});

test("createTranslator returns Chinese labels for AI admin permission packages", () => {
  const t = createTranslator("zh-CN");

  assert.equal(t("nav.ai-admin"), "权限包");
  assert.equal(t("page.aiAdmin"), "权限包审批与验收");
  assert.equal(t("panel.aiAdminPermissionWorkbench"), "权限包审批台");
  assert.equal(t("section.aiAdminApprovalJourney"), "验收证据");
  assert.equal(t("section.aiAdminReadiness"), "环境检查");
  assert.equal(t("section.aiAdminProductionConsole"), "上线状态");
  assert.equal(t("action.runApprovalJourney"), "执行验收");
  assert.equal(t("action.startPermissionApproval"), "发起权限审批");
  assert.equal(t("action.runningApprovalJourney"), "验收中");
  assert.equal(t("action.checkApprovalReadiness"), "检查环境");
  assert.equal(t("text.aiAdminApprovalJourneyCompletion"), "上线进度");
  assert.equal(t("text.aiAdminProductionConsoleHelp"), "按申请、审批、落地、验证和验收汇总上线状态。");
  assert.equal(t("productionConsole.request"), "变更申请");
  assert.equal(t("productionConsole.policyGate"), "策略门禁");
  assert.equal(t("productionConsole.approval"), "审批");
  assert.equal(t("productionConsole.application"), "权限落地");
  assert.equal(t("productionConsole.runtime"), "运行验证");
  assert.equal(t("productionConsole.productionReadiness"), "上线验收");
  assert.equal(t("productionConsole.approvalNotRequired"), "无需审批");
  assert.equal(t("productionConsole.applicationPending"), "待应用权限包");
  assert.equal(t("productionConsole.runtimeEvidence"), "需完成允许/拒绝调用");
  assert.equal(t("readiness.aiAdmin.privateUpstreams.detail"), "本地回环 MCP 目标需要启用本地开发私有上游访问。");
  assert.equal(t("journey.aiAdmin.step.approvedApply"), "已审批权限包应用");
  assert.equal(t("readiness.aiAdmin.subjectHeader.title"), "主体 Header CORS");
  assert.equal(t("section.permissionApplicationEvidence"), "应用证据");
  assert.equal(t("detail.applicationId"), "应用记录 ID");
  assert.equal(t("section.permissionDraft"), "权限变更草案");
  assert.equal(t("section.permissionPolicyGate"), "策略门禁");
  assert.equal(t("section.permissionSimulation"), "发布前模拟");
  assert.equal(t("action.applyPermissionPackage"), "应用权限包");
  assert.equal(t("action.createApprovalRequest"), "发起审批");
  assert.equal(t("section.permissionApprovalRequest"), "审批请求");
  assert.equal(t("section.permissionReviewerQueue"), "审查员队列");
  assert.equal(t("form.approvalReviewer"), "审查员");
  assert.equal(t("action.refreshReviewerQueue"), "刷新队列");
  assert.equal(t("empty.reviewerQueue.detail"), "输入审查员标识后刷新待处理审批。");
  assert.equal(t("message.reviewerQueueLoaded"), "已加载 {count} 个待处理审批。");
  assert.equal(t("section.accessDecisionExplain"), "权限判定说明");
  assert.equal(t("action.explainAccessDecision"), "查看判定");
  assert.equal(t("form.subjectId"), "主体 ID");
  assert.equal(t("empty.accessDecisionExplain.detail"), "选择租户、工作区、调用方、目标和能力后查看判定原因。");
  assert.equal(t("message.accessDecisionExplainLoaded"), "权限判定已加载。");
  assert.equal(t("message.accessDecisionExplainMissingFields"), "请先选择租户、工作区、调用方、目标和能力，再查看权限判定。");
  assert.equal(t("text.nextActions"), "下一步动作");
  assert.equal(t("section.permissionApplicationImpact"), "影响复核");
  assert.equal(t("section.permissionApplicationHealth"), "落地状态");
  assert.equal(t("section.permissionProductionReadiness"), "上线验收");
  assert.equal(t("action.reviewApplicationImpact"), "查看影响");
  assert.equal(t("action.refreshApplicationHealth"), "刷新状态");
  assert.equal(t("action.checkProductionReadiness"), "检查上线验收");
  assert.equal(t("action.exportProductionEvidence"), "导出证据");
  assert.equal(t("action.rehearseApplicationDrift"), "演练漂移");
  assert.equal(t("message.permissionApplicationImpactLoaded"), "影响复核已加载。");
  assert.equal(t("message.permissionApplicationHealthLoaded"), "落地状态已加载。");
  assert.equal(t("message.permissionProductionReadinessLoaded"), "上线验收已加载。");
  assert.equal(t("message.productionEvidenceExported"), "上线证据报告已导出。");
  assert.equal(t("message.productionEvidenceRequiresLiveApi"), "导出上线证据需要实时 API。");
  assert.equal(t("message.permissionProductionReadinessRequiresLiveApi"), "上线验收需要实时 API。");
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
  assert.equal(t("empty.permissionApplicationImpact.detail"), "先查看已应用权限包的影响，再规划任何回滚动作。");
  assert.equal(t("empty.permissionApplicationHealth.detail"), "应用权限包后刷新落地状态。");
  assert.equal(t("empty.remediationActions.detail"), "暂无处置动作。");
  assert.equal(t("value.read_only"), "只读");
  assert.equal(t("value.verify"), "校验");
  assert.equal(t("value.shared_capability_manual_review"), "共享能力需要人工评审，不能自动降级。");
  assert.equal(t("value.disable_instance_assignment"), "先禁用记录的实例分配，缩小调用方访问面。");
  assert.equal(t("value.verify_effective_access"), "手动处置后重新解释有效权限，确认访问已按预期收敛。");
  assert.equal(t("value.manual_review"), "人工评审");
  assert.equal(t("value.investigate"), "调查");
  assert.equal(t("status.directApplyAllowed"), "可直接应用");
  assert.equal(t("status.approvalRequired"), "需要审批");
  assert.equal(t("status.applicationHealthReady"), "健康");
  assert.equal(t("status.applicationHealthDrifted"), "已漂移");
  assert.equal(t("status.applicationHealthNeedsReview"), "需复核");
  assert.equal(t("status.productionReady"), "可上线");
  assert.equal(t("status.productionNeedsReview"), "需复核");
  assert.equal(t("status.productionBlocked"), "阻断");
  assert.equal(t("productionCheck.runtime_allowed_trace_present"), "允许运行证据");
  assert.equal(t("productionCheck.runtime_denied_trace_present"), "拒绝运行证据");
  assert.equal(t("empty.permissionProductionReadiness.detail"), "应用权限包并采集运行、审计和落地证据后检查上线验收。");
  assert.equal(t("status.approvalPending"), "待审批");
  assert.equal(t("message.permissionApprovalPending"), "审批请求仍在待审，请先批准再应用。");
  assert.equal(t("permissionSimulation.guardrailFinance"), "销售只读权限包不包含财务字段访问。");
  assert.equal(t("permissionPolicy.riskApprovalRequired"), "{capability} 是 {risk} 风险能力，需要先审批。");
  assert.equal(t("message.permissionPackageApprovalRequired"), "权限包需要先审批：{detail}。");
  assert.equal(t("message.noMatchingAllowedCapabilities"), "当前目标没有匹配的允许能力。");
});

test("createTranslator falls back to English for missing keys", () => {
  const t = createTranslator("zh-CN");

  assert.equal(t("missing.key", "Fallback"), "Fallback");
});
