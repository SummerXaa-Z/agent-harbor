export type MetricTone = "success" | "warning" | "danger" | "info" | "neutral"

export interface RuntimeEvidenceMetric {
  label: string
  value: string
  detail: string
  tone: MetricTone
}

export function consoleDataSourceLabel(loadError: string | undefined, loadedFromApi?: boolean) {
  if (loadError) {
    return "API error"
  }
  return loadedFromApi ? "Go runtime" : "Fallback dataset"
}

export function runtimeEvidenceMetric(allowedTraceCount: number, deniedTraceCount: number): RuntimeEvidenceMetric {
  const totalTraceCount = allowedTraceCount + deniedTraceCount
  if (totalTraceCount === 0) {
    return {
      label: "Runtime Evidence",
      value: "0",
      detail: "no traces yet",
      tone: "neutral",
    }
  }

  return {
    label: "Runtime Evidence",
    value: String(totalTraceCount),
    detail: `${allowedTraceCount} allowed / ${deniedTraceCount} denied`,
    tone: deniedTraceCount > 0 ? "info" : "success",
  }
}
