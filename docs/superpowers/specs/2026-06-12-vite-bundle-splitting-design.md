# Vite Bundle Splitting Design

## Goal

Remove the current frontend production build chunk-size warning without hiding it, while keeping the control-plane UI behavior and permission workflows unchanged.

## Problem

The frontend currently builds one large JavaScript entry asset, around 611 KB uncompressed. Vite warns because the entry chunk is above the default 500 KB threshold. This is already documented as a public-readiness follow-up, and leaving it unresolved weakens release confidence.

The large chunk comes from bundling React, React DOM, lucide icons, `ConsoleController`, and all control-plane views into one initial asset. The current `frontend/vite.config.ts` does not define any production chunking strategy.

## Product Decision

Treat this as a production-readiness improvement, not a UI rewrite. The user-visible console must keep the same routes, copy, forms, and permission journey. The build should make the existing warning disappear by splitting stable vendor dependencies away from application code.

Do not raise `build.chunkSizeWarningLimit`. A higher limit would silence the warning without reducing the initial JavaScript work.

## Architecture

Use Vite 8 build configuration with `build.rolldownOptions.output` to define explicit chunk groups for stable third-party code:

- `react-vendor`: `react` and `react-dom`.
- `icons-vendor`: `lucide-react`.

Keep app code in the normal entry chunk for this slice. Route-level `React.lazy` can be a later optimization if the application chunk keeps growing, but it has higher UI-state regression risk because `ConsoleController` wires many handlers and handoff contexts.

Add a focused frontend test that reads `frontend/vite.config.ts` as source and verifies:

- the config does not increase `chunkSizeWarningLimit`;
- React, React DOM, and lucide are assigned to explicit vendor chunk groups.

## Verification

Run:

- `pnpm --dir frontend test`
- `pnpm --dir frontend build`
- `make check`
- `make release-check`

The frontend build must emit multiple JavaScript assets and must not show the previous single-entry chunk-size warning.

## Non-Goals

- No backend changes.
- No permission model changes.
- No UI copy or visual layout changes.
- No route-level lazy loading in this slice unless vendor chunking fails to remove the warning.
