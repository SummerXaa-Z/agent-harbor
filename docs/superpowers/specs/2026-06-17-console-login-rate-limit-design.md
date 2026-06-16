# Console Login Rate Limit Design

## Problem

Production console login accepts an administrator key and currently gives immediate feedback for every failed attempt. A browser or script can repeatedly guess keys against `/api/v1/auth/login` without backoff. This is a small but important production security gap now that browser login is a first-class path.

## Decision

Add an in-memory, per-client failure throttle for console login.

- Track failed login attempts by client IP address.
- Allow up to 5 failed attempts inside a 5 minute window.
- On the 6th failed attempt, return `429 RATE_LIMITED` and keep returning 429 until the window expires.
- A successful login clears the failure counter for that client.
- Local unauthenticated development mode bypasses the limiter because it does not use admin key login.

This is intentionally process-local. It raises the floor for single-node and local production deployments without adding persistence, migrations, or external cache requirements.

## Scope

- Add a `TooManyRequests` domain error helper.
- Add rate-limit state and helpers to `internal/httpapi/auth.go`.
- Call the limiter before key verification and record only failed key checks.
- Add backend tests proving the throttle trips, successful login clears the counter, and API key management requests remain unaffected.
- Add changelog notes.

## Non-Goals

- No distributed rate limiting.
- No CAPTCHA or SSO integration.
- No user-configurable thresholds in this slice.
- No UI changes beyond the existing generic login failure message.

## Verification

- Focused `go test ./internal/httpapi -run 'TestConsoleLoginRateLimit|TestConsoleAuthSessionProtectsManagementEndpoints' -count=1`.
- Full `make check` and `make release-check`.
