import { useCallback, useEffect, useState } from "react";
import { fetchAuditEvents, fetchTraces } from "../../api";
import type { AuditEvent, TraceEvent } from "../../types";
import { auditRangeSince, auditTimelineRows, type AuditTimelineRow } from "../model/auditTimeline";

export type AuditRangeKey = "today" | "week" | "month";

export const auditRanges: readonly AuditRangeKey[] = ["today", "week", "month"];

export function normalizeAuditRange(value: string | undefined): AuditRangeKey {
  return auditRanges.includes(value as AuditRangeKey) ? (value as AuditRangeKey) : "week";
}

export interface AdminAuditState {
  failed: boolean;
  loading: boolean;
  raw: { audits: readonly AuditEvent[]; traces: readonly TraceEvent[] };
  reload: () => Promise<void>;
  rows: readonly AuditTimelineRow[];
}

const windowRowLimit = 200;

// One merged audit + trace window for the runtime audit page and the cockpit
// "recent audit" card. P0 gave both endpoints since/limit windows, so the
// range select refetches instead of slicing whatever the console load brought.
export function useAdminAudit(live: boolean, range: AuditRangeKey): AdminAuditState {
  const [raw, setRaw] = useState<{ audits: readonly AuditEvent[]; traces: readonly TraceEvent[] }>({
    audits: [],
    traces: [],
  });
  const [loading, setLoading] = useState(false);
  const [failed, setFailed] = useState(false);

  const reload = useCallback(async () => {
    const since = auditRangeSince(range);
    setLoading(true);
    const [audits, traces] = await Promise.allSettled([
      fetchAuditEvents({ limit: windowRowLimit, since }),
      fetchTraces({ limit: windowRowLimit, since }),
    ]);
    const ok = audits.status === "fulfilled" && traces.status === "fulfilled";
    if (ok) {
      setRaw({ audits: audits.value, traces: traces.value });
    }
    setFailed(!ok);
    setLoading(false);
  }, [range]);

  useEffect(() => {
    if (!live) return;
    void reload();
  }, [live, reload]);

  return { failed, loading, raw, reload, rows: auditTimelineRows(raw.audits, raw.traces) };
}
