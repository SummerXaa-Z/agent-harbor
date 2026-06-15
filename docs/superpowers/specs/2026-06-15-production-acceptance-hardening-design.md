# Production Acceptance Hardening Design

## Context

PR #83 added the production journey checkpoint and merged into `main` as `1ba290d`. The next release-risk items are not new product surfaces. They are production confidence gaps:

- GitHub Dependabot reports two open alerts in `frontend/pnpm-lock.yaml`, both from `esbuild@0.27.7`.
- `vite@8.0.16` accepts `esbuild ^0.27.0 || ^0.28.0`, so the patched `0.28.1` line can be forced without upgrading Vite.
- `make ai-admin-browser-journey` starts the API, real MCP demo, and Vite console, then runs the permission-package scenario, but it does not assert that the web console itself renders the production journey checkpoint or avoids forbidden visible wording.

## Goal

Make the v0.2.0 production acceptance path safer to ship by closing the known frontend dependency alerts and adding a repeatable web-console journey smoke gate that verifies the merged UI actually exposes the production path.

## Non-Goals

- Do not redesign the UI.
- Do not add new product workflows.
- Do not modify backend permission semantics.
- Do not introduce new browser automation dependencies.
- Do not rename internal `evidence` API fields in this slice; only user-facing copy remains guarded by existing tests.

## Approach Options

### Option A: Minimal Security Patch Only

Add a pnpm override for `esbuild@0.28.1`, update the lockfile, and run release gates.

Trade-off: fastest and safest, but it leaves the current "browser journey" naming gap: the script still does not prove UI rendering.

### Option B: Security Patch Plus Lightweight Web Console Smoke

Add the `esbuild` override, then add a small script that starts or reuses the existing local demo stack, fetches the web console, and performs lightweight checks against the served app and route URLs. The check remains dependency-free and complements unit tests with a repeatable release gate.

Trade-off: stronger release confidence without adding Playwright, but it cannot inspect a fully hydrated DOM. It should be named as a smoke gate, not a full visual/browser test.

### Option C: Full Browser Automation Gate

Add Playwright or another browser dependency, run hydrated DOM checks in CI, and capture screenshots.

Trade-off: highest confidence, but it adds dependency weight and a new failure mode while the product is still stabilizing. It is too much for this slice.

## Decision

Use Option B.

This is the right balance for the current phase: fix the concrete security alert, add a repeatable release signal, and keep the branch small. The in-app browser can still be used manually for final review, but the repository gate should stay dependency-light.

## Implementation Design

### Dependency Security

Add a `pnpm.overrides` block to `frontend/package.json`:

```json
"pnpm": {
  "overrides": {
    "esbuild": "0.28.1"
  }
}
```

Then refresh `frontend/pnpm-lock.yaml` with pnpm. Add a frontend test that reads both files and asserts:

- `frontend/package.json` contains the override.
- `frontend/pnpm-lock.yaml` no longer resolves `esbuild@0.27.7`.
- `frontend/pnpm-lock.yaml` resolves `esbuild@0.28.1`.

This makes the Dependabot fix intentional and prevents accidental downgrade.

### Web Console Production Smoke Gate

Add a dependency-free script, `scripts/scenario-web-console-production-journey.sh`, that:

1. Starts API, real MCP, and frontend on isolated configurable ports.
2. Waits for `/healthz`, MCP `/healthz`, and the frontend root.
3. Verifies the served frontend includes the React root.
4. Uses the existing management API and scenario path enough to seed or validate a configured system.
5. Performs route-level smoke checks for:
   - `#getting-started`
   - `#registry`
   - `#ask`
   - `#ai-admin`
   - `#evidence`
6. Confirms the static app bundle includes the production journey component and does not expose visible Chinese forensic wording in translation maps.

The script should be honest about its coverage: it is a production smoke gate, not a full hydrated browser visual test. It should fail fast with logs when a service does not start.

### Makefile Integration

Add:

- `web-console-production-journey` target.
- `frontend-deps` dependency for the new target.
- inclusion in `release-check` after `production-hardening`, so release readiness includes the production UI journey smoke.
- a Makefile target regression assertion in `tests/makefile_targets_test.sh`.

### Documentation

Update:

- `README.md`
- `docs/engineering/release-checklist.md`
- `CHANGELOG.md`

Document that:

- dependency security alerts are closed by the `esbuild` override,
- `make web-console-production-journey` is a lightweight smoke gate,
- hydrated visual/browser verification remains a manual reviewer step unless a future PR adds a real browser automation dependency.

## Testing

Focused:

- `pnpm --dir frontend exec node --test tests/viteConfig.test.mjs`
- `bash tests/makefile_targets_test.sh`
- `bash scripts/scenario-web-console-production-journey.sh`
- `git diff --check`

Full:

- `pnpm --dir frontend test`
- `pnpm --dir frontend build`
- `make check`
- `make release-check`

External:

- Query GitHub Dependabot alerts after push/PR to confirm the `esbuild` alerts close or are at least no longer present on the branch.

## Risks

- `esbuild@0.28.1` is accepted by Vite's dependency range, but the lockfile update must be verified with frontend build.
- A dependency-free smoke script cannot guarantee hydrated DOM rendering. The PR description must not overstate it as visual proof.
- Adding the smoke script to `release-check` increases gate runtime. The script should use isolated ports and fast readiness loops to keep the cost bounded.

## Acceptance Criteria

- GitHub's open `esbuild` alerts are addressed by lockfile resolution to `0.28.1`.
- `make release-check` includes the new production web-console smoke gate and passes locally.
- The new smoke script is documented and can run on isolated ports.
- No backend permission behavior changes.
- No new npm dependencies.
