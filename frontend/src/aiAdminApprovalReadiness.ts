export type AiAdminApprovalReadinessStatus = "pending" | "ok" | "warning" | "error";

export type AiAdminApprovalReadinessKey =
  | "api"
  | "mockMcp"
  | "subjectHeader"
  | "privateUpstreams"
  | "dataSource";

export interface AiAdminApprovalReadinessState {
  api: AiAdminApprovalReadinessStatus;
  mockMcp: AiAdminApprovalReadinessStatus;
  subjectHeader: AiAdminApprovalReadinessStatus;
  privateUpstreams: AiAdminApprovalReadinessStatus;
  dataSource: AiAdminApprovalReadinessStatus;
}

export interface AiAdminApprovalReadinessRow {
  key: AiAdminApprovalReadinessKey;
  status: AiAdminApprovalReadinessStatus;
  titleKey: string;
  detailKey: string;
}

export const defaultAiAdminApprovalReadiness: AiAdminApprovalReadinessState = {
  api: "pending",
  dataSource: "pending",
  mockMcp: "pending",
  privateUpstreams: "warning",
  subjectHeader: "pending",
};

export function aiAdminApprovalReadinessCanRun(state: AiAdminApprovalReadinessState) {
  return state.api === "ok" && state.mockMcp === "ok" && state.subjectHeader === "ok";
}

export function aiAdminApprovalReadinessRows(state: AiAdminApprovalReadinessState): AiAdminApprovalReadinessRow[] {
  return [
    {
      detailKey: "readiness.aiAdmin.api.detail",
      key: "api",
      status: state.api,
      titleKey: "readiness.aiAdmin.api.title",
    },
    {
      detailKey: "readiness.aiAdmin.mockMcp.detail",
      key: "mockMcp",
      status: state.mockMcp,
      titleKey: "readiness.aiAdmin.mockMcp.title",
    },
    {
      detailKey: "readiness.aiAdmin.subjectHeader.detail",
      key: "subjectHeader",
      status: state.subjectHeader,
      titleKey: "readiness.aiAdmin.subjectHeader.title",
    },
    {
      detailKey: "readiness.aiAdmin.privateUpstreams.detail",
      key: "privateUpstreams",
      status: state.privateUpstreams,
      titleKey: "readiness.aiAdmin.privateUpstreams.title",
    },
    {
      detailKey: "readiness.aiAdmin.dataSource.detail",
      key: "dataSource",
      status: state.dataSource,
      titleKey: "readiness.aiAdmin.dataSource.title",
    },
  ];
}
