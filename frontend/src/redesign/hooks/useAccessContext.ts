import { useCallback, useEffect, useMemo, useState } from "react";
import { fetchPermissionPackageApplications } from "../../api";
import type { PermissionPackageApplication } from "../../permissionPackages";
import {
  ACCESS_CONTEXT_STORAGE_KEY,
  accessContextOptions,
  normalizeAccessContext,
  parseStoredAccessContext,
  resolveAccessContext,
  serializeAccessContext,
  type AccessContext,
  type AccessContextSource,
} from "../model/accessContext";
import type { RouteParams } from "../router";
import type { RedesignData } from "./useRedesignData";

export interface AccessContextState {
  applications: PermissionPackageApplication[];
  applicationsLoaded: boolean;
  context: AccessContext;
  options: ReturnType<typeof accessContextOptions>;
  reloadApplications: () => void;
  replace: (context: AccessContext) => void;
  source: AccessContextSource;
  update: (patch: Partial<AccessContext>) => void;
}

function readStoredContext(): AccessContext | null {
  try {
    return parseStoredAccessContext(window.localStorage.getItem(ACCESS_CONTEXT_STORAGE_KEY));
  } catch {
    return null;
  }
}

function writeStoredContext(context: AccessContext) {
  try {
    window.localStorage.setItem(ACCESS_CONTEXT_STORAGE_KEY, serializeAccessContext(context));
  } catch {
    // The page still works for this visit when storage is unavailable.
  }
}

// Views remount per route, so the context resolves again on every visit:
// deep link first, then what the user last worked on, then the most recent
// application. Edits persist so Access query and Request access stay on the same request.
export function useAccessContext(
  data: RedesignData,
  routeParams: RouteParams,
  holderAgentIds?: readonly string[],
): AccessContextState {
  const live = Boolean(data.data?.loadedFromApi);
  const agents = useMemo(() => data.data?.agents ?? [], [data.data]);
  const routeParamsKey = JSON.stringify(routeParams);
  const params = useMemo(() => JSON.parse(routeParamsKey) as RouteParams, [routeParamsKey]);
  const [stored] = useState(readStoredContext);
  const [applications, setApplications] = useState<PermissionPackageApplication[]>([]);
  const [applicationsLoaded, setApplicationsLoaded] = useState(false);
  const [edited, setEdited] = useState<AccessContext | null>(null);
  const [applicationsVersion, setApplicationsVersion] = useState(0);

  useEffect(() => {
    if (!live) return;
    const controller = new AbortController();
    fetchPermissionPackageApplications({ limit: 20 }, "", controller.signal)
      .then((rows) => setApplications(rows))
      .catch(() => {
        if (!controller.signal.aborted) setApplications([]);
      })
      .finally(() => {
        if (!controller.signal.aborted) setApplicationsLoaded(true);
      });
    return () => controller.abort();
  }, [live, applicationsVersion]);

  const resolved = useMemo(
    () => resolveAccessContext({ agents, applications, routeParams: params, stored }),
    [agents, applications, params, stored],
  );
  const context = edited ?? resolved.context;
  const source: AccessContextSource = edited ? "stored" : resolved.source;

  useEffect(() => {
    if (context.callerInstanceId && context.targetId) writeStoredContext(context);
  }, [context]);

  // Written synchronously too, so a hand-off that navigates right away still
  // lands on the next page with this context.
  const replace = useCallback((next: AccessContext) => {
    const normalized = normalizeAccessContext(next);
    writeStoredContext(normalized);
    setEdited(normalized);
  }, []);
  const update = useCallback(
    (patch: Partial<AccessContext>) => setEdited((current) => normalizeAccessContext({ ...(current ?? resolved.context), ...patch })),
    [resolved.context],
  );
  const reloadApplications = useCallback(() => setApplicationsVersion((version) => version + 1), []);
  const options = useMemo(
    () => accessContextOptions(applications, 6, holderAgentIds),
    [applications, holderAgentIds],
  );

  return { applications, applicationsLoaded, context, options, reloadApplications, replace, source, update };
}
