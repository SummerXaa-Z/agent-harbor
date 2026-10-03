import { useCallback, useEffect, useRef, useState } from "react";
import { fetchDailyMetrics } from "../../api";
import type { DailyMetrics } from "../../types";
import { dailyTrend, type DailyTrend } from "../model/dailyTrend";

export interface DailyMetricsState {
  // True when the backend predates metrics_daily_v1 (404) — the trend then
  // shows the upgrade hint instead of an empty chart.
  unsupported: boolean;
  reload: () => Promise<void>;
  trend: DailyTrend | null;
}

const trendDays = 7;

// The 7-day cockpit trend. `days` counts back from today in the user's local
// timezone so "today" means today on the wall clock, not in UTC.
export function useDailyMetrics(live: boolean): DailyMetricsState {
  const [metrics, setMetrics] = useState<DailyMetrics | null>(null);
  const [unsupported, setUnsupported] = useState(false);
  const runIdRef = useRef(0);

  const reload = useCallback(async () => {
    // A slow earlier reload must not overwrite a newer one's result.
    const runId = ++runIdRef.current;
    try {
      const next = await fetchDailyMetrics({ days: trendDays, tzOffsetMinutes: -new Date().getTimezoneOffset() });
      if (runId !== runIdRef.current) return;
      setMetrics(next);
      setUnsupported(false);
    } catch (error) {
      if (runId !== runIdRef.current) return;
      // Older backends answer 404 without the capability; anything else is a
      // real failure and leaves the previous data in place.
      if (error instanceof Error && "status" in error && (error as { status?: number }).status === 404) {
        setUnsupported(true);
      }
    }
  }, []);

  useEffect(() => {
    if (!live) return;
    void reload();
  }, [live, reload]);

  return { reload, trend: dailyTrend(metrics), unsupported };
}
