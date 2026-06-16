export type AdminAccessMutationAction =
  | "create_admin"
  | "rotate_admin_key"
  | "disable_admin";

export type AdminAccessReloadResult =
  | { action: AdminAccessMutationAction; ok: true }
  | { action: AdminAccessMutationAction; error: unknown; ok: false };

export async function reloadAfterAdminAccessMutation({
  action,
  onReload
}: {
  action: AdminAccessMutationAction;
  onReload: () => Promise<void>;
}): Promise<AdminAccessReloadResult> {
  try {
    await onReload();
    return { action, ok: true };
  } catch (error) {
    return { action, error, ok: false };
  }
}

export function adminAccessReloadFailedMessageKey(action: AdminAccessMutationAction) {
  const keys: Record<AdminAccessMutationAction, string> = {
    create_admin: "message.adminAccessCreatedReloadFailed",
    disable_admin: "message.adminAccessDisabledReloadFailed",
    rotate_admin_key: "message.adminAccessRotatedReloadFailed"
  };
  return keys[action];
}
