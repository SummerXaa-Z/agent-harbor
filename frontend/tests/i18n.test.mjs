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
  assert.equal(t("nav.access"), "权限");
  assert.equal(t("page.access"), "租户权限控制台");
  assert.equal(t("metric.runtimeEvidence"), "运行证据");
  assert.equal(t("action.loadProfile"), "加载档案");
  assert.equal(t("panel.coreJourney"), "核心旅程工作台");
  assert.equal(t("action.runCoreJourney"), "跑通核心旅程");
  assert.equal(t("action.resetCoreJourney"), "重置演示会话");
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

  assert.equal(t("nav.ai-admin"), "AI 管理员");
  assert.equal(t("page.aiAdmin"), "AI 权限包工作台");
  assert.equal(t("panel.aiAdminPermissionWorkbench"), "AI 管理员权限包工作台");
  assert.equal(t("section.aiAdminApprovalJourney"), "实时审批旅程");
  assert.equal(t("section.aiAdminReadiness"), "首次运行就绪状态");
  assert.equal(t("action.runApprovalJourney"), "跑通审批旅程");
  assert.equal(t("action.checkApprovalReadiness"), "检查就绪状态");
  assert.equal(t("text.aiAdminApprovalJourneyCompletion"), "审批旅程完成度");
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
  assert.equal(t("section.accessDecisionExplain"), "有效权限解释");
  assert.equal(t("action.explainAccessDecision"), "解释访问");
  assert.equal(t("form.subjectId"), "主体 ID");
  assert.equal(t("empty.accessDecisionExplain.detail"), "选择租户、工作区、调用方、目标和能力后查看原因。");
  assert.equal(t("message.accessDecisionExplainLoaded"), "权限解释已加载。");
  assert.equal(t("text.nextActions"), "下一步动作");
  assert.equal(t("section.permissionApplicationImpact"), "应用影响复盘");
  assert.equal(t("action.reviewApplicationImpact"), "复盘影响");
  assert.equal(t("message.permissionApplicationImpactLoaded"), "应用影响已加载。");
  assert.equal(t("message.permissionApplicationImpactRequiresLiveApi"), "应用影响复盘需要实时 API。");
  assert.equal(t("metric.activeObjects"), "有效对象");
  assert.equal(t("metric.missingObjects"), "缺失对象");
  assert.equal(t("text.rollbackReview"), "回滚评审");
  assert.equal(t("rollbackStep.capabilityManualReview"), "人工复核能力发现状态；共享能力不会被自动降级。");
  assert.equal(t("blocker.missing_created_objects"), "部分已记录授权对象缺失；回滚前需要先调查漂移。");
  assert.equal(t("blocker.inactive_created_objects"), "部分已记录授权对象未启用；需要复核是否已经发生手工变更或部分处置。");
  assert.equal(t("blocker.no_allowed_capabilities"), "该应用没有记录允许能力；不能按自动顺序规划回滚。");
  assert.equal(t("text.remediationPlan"), "处置计划");
  assert.equal(t("text.readOnlyPlan"), "只读计划");
  assert.equal(t("metric.plannedActions"), "计划动作");
  assert.equal(t("text.finalAccessVerification"), "最终访问校验");
  assert.equal(t("text.capabilityReview"), "能力评审");
  assert.equal(t("text.workspaceAssignment"), "工作区分配");
  assert.equal(t("text.instanceAssignment"), "实例分配");
  assert.equal(t("empty.permissionApplicationImpact.detail"), "先复盘已应用权限包的影响，再规划任何回滚动作。");
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
