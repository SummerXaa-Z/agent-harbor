export type ManagementMutationAction =
  | "create_agent"
  | "create_key"
  | "rotate_credential"
  | "create_policy";

export type ManagementMutationRefreshState =
  | { status: "idle" }
  | { action: ManagementMutationAction; status: "refreshing" }
  | { action: ManagementMutationAction; refreshedAt: string; status: "fresh" }
  | { action: ManagementMutationAction; errorMessage: string; status: "stale" };

export type ManagementMutationRefreshResult =
  | { action: ManagementMutationAction; ok: true; refreshedAt: string }
  | { action: ManagementMutationAction; error: unknown; ok: false };

export async function refreshAfterManagementMutation({
  action,
  now = () => new Date(),
  onRefresh
}: {
  action: ManagementMutationAction;
  now?: () => Date;
  onRefresh: () => Promise<void>;
}): Promise<ManagementMutationRefreshResult> {
  try {
    await onRefresh();
    return { action, ok: true, refreshedAt: now().toISOString() };
  } catch (error) {
    return { action, error, ok: false };
  }
}

export function managementMutationSuccessMessageKey(action: ManagementMutationAction) {
  const keys: Record<ManagementMutationAction, string> = {
    create_agent: "message.agentCreated",
    create_key: "message.keyCreated",
    create_policy: "message.policyCreated",
    rotate_credential: "message.credentialRotated"
  };
  return keys[action];
}

export function managementMutationRefreshFailedMessageKey(action: ManagementMutationAction) {
  const keys: Record<ManagementMutationAction, string> = {
    create_agent: "message.agentCreatedRefreshFailed",
    create_key: "message.keyCreatedRefreshFailed",
    create_policy: "message.policyCreatedRefreshFailed",
    rotate_credential: "message.credentialRotatedRefreshFailed"
  };
  return keys[action];
}

