export type ConsoleDataStatus = "loading" | "live" | "sample" | "error";

export interface ConsoleDataStatusInput {
  hasData: boolean;
  hasError: boolean;
  loadedFromApi?: boolean;
  loading: boolean;
}

// Sample data is never shown silently: `loadConsoleData` falls back to sample
// rows on network failures, so that state gets its own status and banner.
// A failed refresh keeps the previous data and its status; the toast reports it.
export function consoleDataStatus({ hasData, hasError, loadedFromApi, loading }: ConsoleDataStatusInput): ConsoleDataStatus {
  if (hasData) return loadedFromApi ? "live" : "sample";
  if (hasError && !loading) return "error";
  return "loading";
}

export const metricsDailyCapability = "metrics_daily_v1";
export const targetProbeCapability = "target_probe_v1";

export function systemCapabilitySet(info?: { capabilities?: unknown } | null): ReadonlySet<string> {
  const capabilities = Array.isArray(info?.capabilities) ? info.capabilities : [];
  return new Set(capabilities.filter((value): value is string => typeof value === "string" && value.trim() !== ""));
}
