import { useCallback, useEffect, useMemo, useRef, useState, useSyncExternalStore } from "react";
import {
  fetchAgents,
  fetchAgentKeys,
  fetchPermissionPackageApplications,
  fetchPermissionPackageApprovalRequests,
} from "../../api";
import type { Agent, AgentKey, ConsoleSession } from "../../types";
import type { PermissionPackageApprovalRequest } from "../../permissionPackages";
import { usePermissionCatalog } from "./usePermissionCatalog";
import { latestEnvCheckSnapshot, registeredMcpTargets, subscribeEnvCheckSnapshot } from "../model/envChecks";
import {
  deriveNotifications,
  effectiveReadIds,
  NOTIFICATION_STORAGE_KEY,
  parseReadState,
  serializeReadState,
  unreadCounts,
  type NotificationItem,
  type NotificationReadState,
} from "../model/notifications";
import type { RedesignData } from "./useRedesignData";

export interface NotificationsState {
  approvals: readonly PermissionPackageApprovalRequest[];
  items: readonly NotificationItem[];
  markAllRead: (surface: "admin" | "user") => void;
  markRead: (dedupeId: string) => void;
  unread: { admin: number; user: number };
}

const pollIntervalMs = 15_000;

// Notifications (plan §8; phase 2 adds token lifecycle and live structural
// environment rows): the 15s poll fetches approvals, agents, agent keys, and
// applications in parallel (each source fails independently — a restricted
// session simply loses that source's items) and pauses while the tab is
// hidden; probe-dependent environment failures still come from the latest
// env-check snapshot instead of re-probing. Read state persists in
// localStorage and syncs across tabs through the storage event.
export function useNotifications(data: RedesignData, session: ConsoleSession | null): NotificationsState {
  const catalog = usePermissionCatalog(true);
  const [approvals, setApprovals] = useState<readonly PermissionPackageApprovalRequest[]>([]);
  const [agents, setAgents] = useState<readonly Agent[]>([]);
  const [keys, setKeys] = useState<readonly AgentKey[]>([]);
  const [applicationCount, setApplicationCount] = useState(0);
  const [readState, setReadState] = useState<NotificationReadState>(() =>
    parseReadState(readStored(), new Date().toISOString()),
  );
  const pollTimer = useRef<number | null>(null);
  const runIdRef = useRef(0);

  const reload = useCallback(async () => {
    const runId = ++runIdRef.current;
    const [approvalResult, agentResult, keyResult, applicationResult] = await Promise.allSettled([
      fetchPermissionPackageApprovalRequests({ limit: 100 }),
      fetchAgents(),
      fetchAgentKeys(),
      fetchPermissionPackageApplications({ limit: 5 }),
    ]);
    if (runId !== runIdRef.current) return;
    if (approvalResult.status === "fulfilled") setApprovals(approvalResult.value);
    if (agentResult.status === "fulfilled") setAgents(agentResult.value);
    if (keyResult.status === "fulfilled") setKeys(keyResult.value);
    if (applicationResult.status === "fulfilled") setApplicationCount(applicationResult.value.length);
    // Rejections are per-source (a forbidden or unreachable endpoint): the
    // shell keeps working with whatever sources are available.
  }, []);

  useEffect(() => {
    void reload();
    const start = () => {
      if (pollTimer.current !== null) return;
      pollTimer.current = window.setInterval(() => {
        if (document.visibilityState === "hidden") return;
        void reload();
      }, pollIntervalMs);
    };
    const stop = () => {
      if (pollTimer.current === null) return;
      window.clearInterval(pollTimer.current);
      pollTimer.current = null;
    };
    const onVisibility = () => {
      if (document.visibilityState === "visible") {
        void reload();
        start();
      } else {
        stop();
      }
    };
    start();
    document.addEventListener("visibilitychange", onVisibility);
    return () => {
      stop();
      document.removeEventListener("visibilitychange", onVisibility);
    };
  }, [reload]);

  useEffect(() => {
    const onStorage = (event: StorageEvent) => {
      if (event.key !== NOTIFICATION_STORAGE_KEY) return;
      setReadState(parseReadState(readStored(), new Date().toISOString()));
    };
    window.addEventListener("storage", onStorage);
    return () => window.removeEventListener("storage", onStorage);
  }, []);

  // Subscribed so a finished env check re-derives notifications immediately
  // instead of waiting for the next approval poll to change a dependency.
  const envSnapshot = useSyncExternalStore(subscribeEnvCheckSnapshot, latestEnvCheckSnapshot);

  const items = useMemo(() => {
    const polledAgents = agents.length > 0 ? agents : (data.data?.agents ?? []);
    return deriveNotifications({
      agents: polledAgents,
      approvals,
      capabilities: data.data?.capabilities ?? [],
      envRows: envSnapshot?.rows ?? [],
      keys,
      now: Date.now(),
      sessionActor: session?.actor ?? null,
      structuralEnv: {
        applicationCount,
        registeredTargetCount: registeredMcpTargets(polledAgents).length,
      },
      templates: catalog.templates,
    });
  }, [agents, approvals, applicationCount, catalog.templates, data.data, envSnapshot, keys, session]);

  const read = useMemo(() => effectiveReadIds(items, readState), [items, readState]);
  const unread = useMemo(() => unreadCounts(items, read), [items, read]);

  const persist = useCallback((next: NotificationReadState) => {
    setReadState(next);
    try {
      window.localStorage.setItem(NOTIFICATION_STORAGE_KEY, serializeReadState(next));
    } catch {
      // Storage may be unavailable (private mode); reads still work for this tab.
    }
  }, []);

  const markRead = useCallback(
    (dedupeId: string) => {
      if (readState.readIds.includes(dedupeId)) return;
      persist({ ...readState, readIds: [...readState.readIds, dedupeId] });
    },
    [persist, readState],
  );

  const markAllRead = useCallback(
    (surface: "admin" | "user") => {
      const additions = items
        .filter((item) => item.surface === surface && !read.has(item.dedupeId))
        .map((item) => item.dedupeId);
      if (additions.length === 0) return;
      persist({ ...readState, readIds: [...readState.readIds, ...additions] });
    },
    [items, persist, read, readState],
  );

  return { approvals, items, markAllRead, markRead, unread };
}

function readStored(): string | null {
  try {
    return window.localStorage.getItem(NOTIFICATION_STORAGE_KEY);
  } catch {
    return null;
  }
}
