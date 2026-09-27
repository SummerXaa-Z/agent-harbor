import type { Translator } from "../../consolePresenters.ts";
import { tx } from "../../localizedMessages.ts";
import type { DailyMetrics } from "../../types.ts";

export interface DailyTrendPoint {
  auditEvents: number;
  calls: number;
  date: string;
  // Null on days with no calls: the rate is undefined, not zero.
  denyRate: number | null;
}

export interface DailyTrend {
  generatedAt: string;
  points: DailyTrendPoint[];
  totals: { auditEvents: number; calls: number; denied: number };
  truncated: boolean;
}

// `metrics/daily` already pads missing days with zero buckets; the model only
// re-shapes them so the chart and the summary read the same series.
export function dailyTrend(metrics: DailyMetrics | null | undefined): DailyTrend | null {
  if (!metrics) return null;
  const points = (Array.isArray(metrics.buckets) ? metrics.buckets : []).map((bucket) => ({
    auditEvents: bucket.auditEvents,
    calls: bucket.calls,
    date: bucket.date,
    denyRate: bucket.denyRate,
  }));
  return {
    generatedAt: metrics.generatedAt,
    points,
    totals: {
      auditEvents: metrics.totals.auditEvents,
      calls: metrics.totals.calls,
      denied: metrics.totals.deniedCalls,
    },
    truncated: metrics.truncated,
  };
}

// An explicit empty state beats a chart of zero-height bars: "no traffic" and
// "cannot read metrics" must not look the same.
export function trendHasData(trend: DailyTrend | null): boolean {
  if (!trend) return false;
  return trend.points.some((point) => point.calls > 0 || point.auditEvents > 0);
}

export type TrendDirection = "up" | "down" | "flat";

// Compares the last day against the first with a 10% dead zone, so a single
// noisy day does not turn a flat week into a "rise".
export function trendDirection(trend: DailyTrend): TrendDirection {
  const withCalls = trend.points.filter((point) => point.calls > 0);
  if (withCalls.length < 2) return "flat";
  const first = withCalls[0].calls;
  const last = withCalls[withCalls.length - 1].calls;
  if (last > first * 1.1 + 1) return "up";
  if (last < first * 0.9 - 1) return "down";
  return "flat";
}

export interface TrendPeak {
  date: string;
  denyRate: number;
}

export function trendDenyPeak(trend: DailyTrend): TrendPeak | null {
  let peak: TrendPeak | null = null;
  for (const point of trend.points) {
    if (point.denyRate === null) continue;
    if (!peak || point.denyRate > peak.denyRate) peak = { date: point.date, denyRate: point.denyRate };
  }
  return peak;
}

export function trendPeakDayLabel(date: string): string {
  const parsed = new Date(`${date}T00:00:00`);
  if (Number.isNaN(parsed.getTime())) return date;
  return `${String(parsed.getMonth() + 1).padStart(2, "0")}-${String(parsed.getDate()).padStart(2, "0")}`;
}

// One data-generated sentence (plan 13.14): totals, direction, and the deny
// peak day. Never a canned "example data" line.
export function trendSummary(t: Translator, trend: DailyTrend): string | null {
  if (!trendHasData(trend)) return null;
  const direction = t(`rd.trend.dir.${trendDirection(trend)}`);
  const peak = trendDenyPeak(trend);
  if (trend.totals.denied === 0 && !peak) {
    return tx(t, "rd.trend.summaryNoDeny", { auditEvents: trend.totals.auditEvents, calls: trend.totals.calls });
  }
  const summary = tx(t, "rd.trend.summary", {
    auditEvents: trend.totals.auditEvents,
    calls: trend.totals.calls,
    denied: trend.totals.denied,
    direction,
  });
  if (!peak) return summary;
  return `${summary}${tx(t, "rd.trend.summaryPeak", {
    peakDay: trendPeakDayLabel(peak.date),
    peakRate: Math.round(peak.denyRate * 100),
  })}`;
}
