# Console Session Expiry Guard Design

## Context

Browser console sessions include `expiresAt`, but the frontend did not act on it. An administrator could keep the console open past session expiry and only discover the expired session during the next management write. Auth responses also lacked explicit no-store cache controls.

## Decision

Use the existing `expiresAt` contract to schedule a client-side session refresh at expiry. If the refreshed session is unauthenticated, return the operator to the login state with a localized expired-session message. Mark auth session, login, and logout responses as `Cache-Control: no-store` so browser/proxy caches do not reuse stale authentication state.

## Behavior

- `GET /api/v1/auth/session`, `POST /api/v1/auth/login`, and `POST /api/v1/auth/logout` emit `Cache-Control: no-store`.
- The frontend computes the remaining lifetime from `ConsoleSession.expiresAt`.
- Authenticated production sessions schedule a one-shot refresh at expiry.
- Local development sessions with `requiresLogin=false` do not schedule an expiry guard.
- If the refresh returns unauthenticated, the console shows the login screen with `error.consoleSessionExpired`.
- If the refresh fails for network/API reasons, the console falls back to the existing session-unavailable message.

## Non-Goals

- No sliding sessions or automatic renewal.
- No background polling before expiry.
- No change to the signed session TTL.
- No persistent session store.

## Verification

- Backend test proves auth responses are no-store.
- Frontend pure function tests cover future, expired, missing, and invalid timestamps.
- Frontend wiring test proves the auth hook schedules timeout cleanup and exposes bilingual expired-session copy.
- Full `make check` and `make release-check` must pass before PR.
