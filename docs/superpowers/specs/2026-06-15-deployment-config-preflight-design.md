# Deployment Config Preflight Design

## Context

AgentHarbor now has fail-closed management authentication, HttpOnly console sessions, private-upstream defaults, production hardening checks, and a web console production journey gate. The next production-risk gap is deployment configuration clarity: operators can still start a deployment-shaped environment with development-only flags or weak session-secret posture and only discover the issue indirectly.

## Approaches Considered

1. **Hard fail on unsafe process startup.** This is strict, but it risks breaking `make demo`, local scenario scripts, and developer-preview evaluation because AgentHarbor intentionally supports explicit development flags.
2. **HTTP admin endpoint for configuration health.** This is useful later, but it exposes another management surface and requires deciding how much config detail is safe to return.
3. **Pure backend preflight plus release-gate script coverage.** This gives deterministic safety checks, keeps the logic testable without network calls, and lets release validation fail on production-blocking settings without changing local demo behavior.

The selected design is approach 3.

## Design

Add a pure app-level deployment preflight model that evaluates environment-derived deployment posture. The first slice is intentionally small and security-focused:

- `AGENT_HARBOR_DEPLOYMENT_MODE=production` enables production preflight semantics.
- Production mode must block `AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true`.
- Production mode must block `AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true`.
- Production mode must require either `AGENT_HARBOR_ADMIN_KEY` or `AGENT_HARBOR_ADMIN_IDENTITIES`.
- Production mode should warn when `AGENT_HARBOR_SESSION_SECRET` is absent, because current runtime can derive a secret from admin credentials, but deployment handoff should prefer an explicit stable high-entropy secret.
- Production mode with PostgreSQL must continue to require `AGENT_HARBOR_CREDENTIAL_KEY`; existing credential-key validation remains the source of truth.

The preflight returns structured checks with a code, severity, status, and message. `app.New` will run the preflight and fail only when a blocking production check fails. Development mode remains unchanged, including explicit unauthenticated local development.

## Files

- `internal/app/app.go`: add deployment-mode parsing and pure preflight helpers.
- `internal/app/app_test.go`: add red/green coverage for production blocking checks and warning checks.
- `scripts/scenario-production-hardening.sh`: verify production mode fails closed for development flags and still starts with a production-safe minimal configuration.
- `README.md`, `docs/engineering/release-checklist.md`, `CHANGELOG.md`: document the new production configuration preflight in English and Simplified Chinese.

## Non-Goals

- No frontend UI changes in this slice.
- No new environment variable framework.
- No new dependencies.
- No change to permission package approval, runtime authorization, tenant hierarchy, or MCP routing semantics.
- No public config endpoint yet.

## Testing

- Focused app tests cover production/development mode behavior.
- `make production-hardening` covers runtime startup behavior.
- `make check` and `make release-check` remain the final gates.

## Review Notes

The design keeps production safety enforceable without making developer-preview evaluation harder. The key product choice is that production mode becomes explicit through `AGENT_HARBOR_DEPLOYMENT_MODE=production`; without that variable, existing local behavior and release gates remain compatible.
