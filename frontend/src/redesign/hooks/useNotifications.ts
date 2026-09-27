import { useCallback, useEffect, useMemo, useRef, useState, useSyncExternalStore } from "react";
import { fetchPermissionPackageApprovalRequests } from "../../api";
import type { PermissionPackageApprovalRequest } from "../../permissionPackages";
import type { ConsoleSession } from "../../types";
import { usePermissionCatalog } from "./usePermissionCatalog";
import { latestEnvCheckSnapshot, subscribeEnvCheckSnapshot } from "../model/envChecks";
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

// First-phase notifications (plan §8): the approval list is the source of
// truth (newest first, carries reviewComment), polled every 15s and paused
// while the tab is hidden; environment failures come from the latest env-check
// snapshot instead of re-probing. Read state persists in localStorage and
// syncs across tabs through the storage event.
export function useNotifications(data: RedesignData, session: ConsoleSession | null): NotificationsState {
  const catalog = usePermissionCatalog(true);
  const [approvals, setApprovals] = useState<readonly PermissionPackageApprovalRequest[]>([]);
  const [readState, setReadState] = useState<NotificationReadState>(() =>
    parseReadState(readStored(), new Date().toISOString()),
  );
  const pollTimer = useRef<number | null>(null);
  const runIdRef = useRef(0);

  const reload = useCallback(async () => {
    const runId = ++runIdRef.current;
    try {
      const rows = await fetchPermissionPackageApprovalRequests({ limit: 100 });
      if (runId === runIdRef.current) setApprovals(rows);
    } catch {
      // Unreachable or forbidden (a non-admin session): no notifications to
      // derive from, the shell keeps working without them.
    }
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

  const items = useMemo(
    () =>
      deriveNotifications({
        agents: data.data?.agents ?? [],
        approvals,
        capabilities: data.data?.capabilities ?? [],
        envRows: envSnapshot?.rows ?? [],
        sessionActor: session?.actor ?? null,
        templates: catalog.templates,
      }),
    [approvals, catalog.templates, data.data, envSnapshot, session],
  );

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
