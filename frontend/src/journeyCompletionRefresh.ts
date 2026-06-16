export type JourneyCompletionRefreshResult<T> =
  | { ok: true; value: T }
  | { error: unknown; ok: false };

export type JourneyCompletionKind = "ai_admin_approval" | "core_journey";

export async function refreshAfterJourneyCompletion<T>({
  onRefresh
}: {
  onRefresh: () => Promise<T>;
}): Promise<JourneyCompletionRefreshResult<T>> {
  try {
    return { ok: true, value: await onRefresh() };
  } catch (error) {
    return { error, ok: false };
  }
}

export function journeyCompletionRefreshFailedMessageKey(kind: JourneyCompletionKind) {
  const keys: Record<JourneyCompletionKind, string> = {
    ai_admin_approval: "message.aiAdminApprovalJourneyCompleteRefreshFailed",
    core_journey: "message.coreJourneyCompleteRefreshFailed"
  };
  return keys[kind];
}
