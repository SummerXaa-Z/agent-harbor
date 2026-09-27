import { useEffect, useRef, useState } from "react";
import { fetchTenantPermissionCenter, loadTenantAccessProfile } from "../../api";
import type { TenantAccessProfileData, TenantPermissionCenterResponse } from "../../types";

export interface TenantDetailState {
  center: TenantPermissionCenterResponse | null;
  centerFailed: boolean;
  loading: boolean;
  profile: TenantAccessProfileData | null;
  profileFailed: boolean;
}

// Feeds the tenant detail tabs (D2): the permission-center snapshot and the
// access profile render the three-layer grant chain. Both endpoints fall
// back to sample data inside the API layer when unreachable.
export function useTenantDetail(live: boolean, tenantId: string): TenantDetailState {
  const [state, setState] = useState<TenantDetailState>({
    center: null,
    centerFailed: false,
    loading: false,
    profile: null,
    profileFailed: false
  });
  const runIdRef = useRef(0);

  useEffect(() => {
    const trimmed = tenantId.trim();
    const runId = ++runIdRef.current;
    if (!live || !trimmed) {
      setState({
        center: null,
        centerFailed: false,
        loading: false,
        profile: null,
        profileFailed: false
      });
      return;
    }
    const controller = new AbortController();
    setState((current) => ({ ...current, loading: true }));
    void (async () => {
      const [centerResult, profileResult] = await Promise.allSettled([
        fetchTenantPermissionCenter(trimmed, undefined, undefined, controller.signal),
        loadTenantAccessProfile(trimmed, undefined, {}, controller.signal)
      ]);
      if (runId !== runIdRef.current) return;
      setState({
        center: centerResult.status === "fulfilled" ? centerResult.value : null,
        centerFailed: centerResult.status === "rejected",
        loading: false,
        profile: profileResult.status === "fulfilled" ? profileResult.value : null,
        profileFailed: profileResult.status === "rejected"
      });
    })();
    return () => controller.abort();
  }, [live, tenantId]);

  return state;
}
