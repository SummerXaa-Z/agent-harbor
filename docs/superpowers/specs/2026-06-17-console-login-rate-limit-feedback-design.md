# Console Login Rate Limit Feedback Design

## Context

AgentHarbor now rate-limits repeated failed browser console admin-key logins per client. The first security slice returned `RATE_LIMITED` with HTTP 429, but the browser could only show the generic sign-in failure message. That makes a deliberate security lockout look like a wrong key or a broken console.

## Decision

Expose the lockout retry window through the standard `Retry-After` response header and let the web console translate `RATE_LIMITED` into explicit retry guidance. Keep the existing lockout policy unchanged: five failed attempts per client within a five-minute process-local window, successful login clears the counter, and authenticated API key automation bypasses the browser login throttle.

## Behavior

- When login is blocked, `/api/v1/auth/login` returns HTTP 429, error code `RATE_LIMITED`, and `Retry-After` with the remaining retry window in seconds.
- The frontend API client preserves backend error code and retry timing on `ApiRequestError`.
- The console login hook maps `RATE_LIMITED` to `error.consoleLoginRateLimited` with `{seconds}` params.
- English and Simplified Chinese copy explain that there were too many failed attempts and the user should retry later.

## Non-Goals

- No new dependency.
- No persistent or distributed rate-limit store.
- No change to admin key verification semantics.
- No countdown timer or automatic retry.
- No change to API key management authentication.

## Verification

- Backend HTTP handler test must fail before the header is implemented and then pass with `Retry-After: 300`.
- Frontend tests must prove retry metadata is preserved and localized login guidance is wired.
- Full release gates must pass before PR.
