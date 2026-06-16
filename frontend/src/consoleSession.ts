export function sessionExpiryDelayMs(expiresAt?: string, nowMs: number = Date.now()): number | null {
  if (!expiresAt) return null
  const expiresAtMs = Date.parse(expiresAt)
  if (!Number.isFinite(expiresAtMs)) return null
  return Math.max(0, expiresAtMs - nowMs)
}
