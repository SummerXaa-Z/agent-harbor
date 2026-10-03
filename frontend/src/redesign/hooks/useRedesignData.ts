import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { fetchSystemInfo, loadConsoleData } from "../../api";
import type { SystemInfo } from "../../systemInfoContract";
import type { ConsoleData } from "../../types";
import { consoleDataStatus, systemCapabilitySet, type ConsoleDataStatus } from "../model/dataStatus";

interface RedesignDataState {
  data: ConsoleData | null;
  error: unknown;
  failed: boolean;
  loading: boolean;
  systemInfo: SystemInfo | null;
}

export interface RedesignData extends RedesignDataState {
  capabilities: ReadonlySet<string>;
  // Resolves true only when live data arrived, so callers never announce a
  // refresh that fell back to sample rows or failed.
  reload: () => Promise<boolean>;
  status: ConsoleDataStatus;
}

const initialState: RedesignDataState = {
  data: null,
  error: null,
  failed: false,
  loading: false,
  systemInfo: null,
};

// Loads whenever access becomes ready (first visit, or signing in again) and
// drops everything on sign-out so no previous session's data lingers.
export function useRedesignData(enabled: boolean): RedesignData {
  const [state, setState] = useState<RedesignDataState>(initialState);
  const mountedRef = useRef(true);
  const requestRef = useRef(0);
  const previousDataRef = useRef<ConsoleData | null>(null);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const reload = useCallback(async () => {
    const requestId = ++requestRef.current;
    setState((current) => ({ ...current, loading: true }));
    // Only real rows are eligible as fallback carriers, so samples never
    // replace data that was already live (or already retained) mid-session.
    const previous = previousDataRef.current;
    const previousForFallback =
      previous && (previous.loadedFromApi || previous.retainedFromPrevious) ? previous : undefined;
    const [dataResult, infoResult] = await Promise.allSettled([
      loadConsoleData(undefined, {}, undefined, previousForFallback),
      fetchSystemInfo(),
    ]);
    if (!mountedRef.current || requestId !== requestRef.current) return false;
    setState((current) => {
      const data = dataResult.status === "fulfilled" ? dataResult.value : current.data;
      previousDataRef.current = data;
      return {
        data,
        error: dataResult.status === "rejected" ? dataResult.reason : null,
        failed: dataResult.status === "rejected",
        loading: false,
        systemInfo: infoResult.status === "fulfilled" ? infoResult.value : current.systemInfo,
      };
    });
    return dataResult.status === "fulfilled" && dataResult.value.loadedFromApi;
  }, []);

  useEffect(() => {
    if (enabled) {
      void reload();
      return;
    }
    requestRef.current += 1;
    previousDataRef.current = null;
    setState((current) => (current === initialState ? current : initialState));
  }, [enabled, reload]);

  const capabilities = useMemo(() => systemCapabilitySet(state.systemInfo), [state.systemInfo]);
  const status = consoleDataStatus({
    hasData: state.data !== null,
    hasError: state.failed,
    loadedFromApi: state.data?.loadedFromApi,
    loading: state.loading,
    retainedFromPrevious: state.data?.retainedFromPrevious,
  });

  return { ...state, capabilities, reload, status };
}
