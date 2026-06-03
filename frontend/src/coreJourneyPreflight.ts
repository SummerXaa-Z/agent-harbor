export type CoreJourneyPreflightStatus = "pending" | "ok" | "warning" | "error";

export interface CoreJourneyPreflightState {
  api: CoreJourneyPreflightStatus;
  mockMcp: CoreJourneyPreflightStatus;
  privateUpstreams: CoreJourneyPreflightStatus;
}

export interface CoreJourneyPreflightRow {
  key: keyof CoreJourneyPreflightState;
  status: CoreJourneyPreflightStatus;
  titleKey: string;
  detailKey: string;
}

export const defaultCoreJourneyPreflight: CoreJourneyPreflightState = {
  api: "pending",
  mockMcp: "pending",
  privateUpstreams: "warning",
};

export function coreJourneyPreflightCanRun(state: CoreJourneyPreflightState) {
  return state.api === "ok" && state.mockMcp === "ok";
}

export function coreJourneyPreflightRows(state: CoreJourneyPreflightState): CoreJourneyPreflightRow[] {
  return [
    {
      key: "api",
      status: state.api,
      titleKey: "preflight.api.title",
      detailKey: "preflight.api.detail",
    },
    {
      key: "mockMcp",
      status: state.mockMcp,
      titleKey: "preflight.mockMcp.title",
      detailKey: "preflight.mockMcp.detail",
    },
    {
      key: "privateUpstreams",
      status: state.privateUpstreams,
      titleKey: "preflight.privateUpstreams.title",
      detailKey: "preflight.privateUpstreams.detail",
    },
  ];
}
