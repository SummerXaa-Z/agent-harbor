import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { DailyMetrics } from "../../src/api";
import { useDailyMetrics } from "../../src/redesign/hooks/useDailyMetrics";

const fetchDailyMetrics = vi.hoisted(() => vi.fn());
vi.mock("../../src/api", () => ({ fetchDailyMetrics }));

function metrics(calls: number): DailyMetrics {
  return {
    days: 7,
    tzOffsetMinutes: 0,
    from: "2026-10-28",
    to: "2026-11-03",
    generatedAt: "2026-11-03T00:00:00Z",
    truncated: false,
    totals: { calls, allowedCalls: calls, deniedCalls: 0, auditEvents: 0 },
    buckets: [],
  };
}

beforeEach(() => {
  fetchDailyMetrics.mockReset();
});

describe("useDailyMetrics", () => {
  it("does not fetch while the console data is not live", () => {
    renderHook(() => useDailyMetrics(false));
    expect(fetchDailyMetrics).not.toHaveBeenCalled();
  });

  it("loads the trend for a 7-day local-time window once live", async () => {
    fetchDailyMetrics.mockResolvedValue(metrics(5));
    const { result } = renderHook(() => useDailyMetrics(true));
    await waitFor(() => expect(result.current.trend?.totals.calls).toBe(5));
    expect(result.current.unsupported).toBe(false);
    const params = fetchDailyMetrics.mock.calls[0][0] as { days: number };
    expect(params.days).toBe(7);
  });

  it("ignores a slow earlier reload that lands after a newer one", async () => {
    // The runId guard (audit LOW#4): the mount reload stalls, a manual
    // reload finishes first, and the stale response must not overwrite it.
    const deferred: Array<(value: DailyMetrics) => void> = [];
    fetchDailyMetrics.mockImplementation(
      () => new Promise<DailyMetrics>((resolve) => { deferred.push(resolve); }),
    );
    const { result } = renderHook(() => useDailyMetrics(true));
    await act(async () => { void result.current.reload(); });
    expect(deferred).toHaveLength(2);

    await act(async () => { deferred[1](metrics(9)); });
    expect(result.current.trend?.totals.calls).toBe(9);

    await act(async () => { deferred[0](metrics(2)); });
    expect(result.current.trend?.totals.calls).toBe(9);
  });

  it("marks the backend unsupported on a 404 without surfacing a trend", async () => {
    fetchDailyMetrics.mockRejectedValue(Object.assign(new Error("not found"), { status: 404 }));
    const { result } = renderHook(() => useDailyMetrics(true));
    await waitFor(() => expect(result.current.unsupported).toBe(true));
    expect(result.current.trend).toBeNull();
  });

  it("keeps the previous trend when a reload fails for another reason", async () => {
    fetchDailyMetrics.mockResolvedValueOnce(metrics(5));
    const { result } = renderHook(() => useDailyMetrics(true));
    await waitFor(() => expect(result.current.trend?.totals.calls).toBe(5));

    fetchDailyMetrics.mockRejectedValue(Object.assign(new Error("boom"), { status: 500 }));
    await act(async () => { await result.current.reload(); });
    expect(result.current.trend?.totals.calls).toBe(5);
    expect(result.current.unsupported).toBe(false);
  });
});
