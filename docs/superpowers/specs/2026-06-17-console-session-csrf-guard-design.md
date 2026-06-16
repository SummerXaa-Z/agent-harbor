# Console Session CSRF Guard Design

## Problem

AgentHarbor now supports browser login with an HttpOnly `agent_harbor_session` cookie. The cookie is `SameSite=Lax`, which is useful, but production management writes still rely on a browser-carried credential. A malicious cross-site request should not be able to mutate Resource Management, permission packages, tenant permissions, or administrator identities just because the browser has an active console session.

## Decision

Add a session-bound CSRF token for console sessions.

- Login and session status responses return `csrfToken` when the user is authenticated through a browser console session.
- The frontend stores the token in memory and automatically sends `X-AgentHarbor-CSRF` on POST, PATCH, and DELETE requests.
- The server requires the header only when authentication is satisfied by the console session cookie.
- Requests authenticated by `X-Admin-Key`, Bearer Agent keys, local unauthenticated development mode, or public read endpoints do not require the CSRF header.

The token is derived from the signed session cookie using the existing session secret. It is not persisted and does not require a database migration.

## Scope

- Add `csrfToken` to the console session API response.
- Add `X-AgentHarbor-CSRF` to local CORS allowed headers.
- Enforce CSRF on unsafe management methods when the console session cookie is used for admin authentication.
- Update frontend API request plumbing to cache the token from session/login responses and attach it to mutating requests.
- Add backend and frontend regression tests.

## Non-Goals

- No change to the admin key or managed administrator identity model.
- No CSRF requirement for API key or Bearer-token automation.
- No persistent CSRF token table.
- No UI redesign.

## Error Handling

Missing or invalid CSRF headers return `403` using the existing permission-denied envelope. The message is deliberately generic: `console session csrf token is required`.

## Verification

- Backend tests prove cookie-authenticated POST without CSRF is rejected, valid CSRF succeeds, GET session remains readable, and `X-Admin-Key` writes are unaffected.
- Frontend tests prove API requests expose `csrfToken`, keep session cookies, and attach `X-AgentHarbor-CSRF` for mutations.
- Full `make check` and `make release-check` remain green.
