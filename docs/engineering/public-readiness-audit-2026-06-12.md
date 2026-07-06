# Public Readiness Audit - 2026-06-12

## Scope

This audit checks whether a first-time public evaluator can clone AgentHarbor, run the local gates, start the demo console, and understand the first operator journey without hidden setup knowledge.

Baseline:

- Public repository: `https://github.com/SummerXaa-Z/agent-harbor`
- Fresh clone commit: `f5664a3`
- Audit branch base: `f5664a3`
- Temporary clone: `/tmp/agent-harbor-public-readiness`

Out of scope:

- New product features.
- Backend behavior changes.
- Dependency updates from Dependabot PRs #72 and #73.

## Fresh Clone Results

From `/tmp/agent-harbor-public-readiness`:

| Check | Result | Notes |
| --- | --- | --- |
| `git status --short --branch` | Pass | Clean `main...origin/main` checkout. |
| `make check` | Pass | Frontend tests reported `205` passing tests. Frontend build completed with the existing Vite chunk-size warning. |
| `make release-check` | Pass | Full release gate completed from the clean clone. Frontend build repeated the same non-blocking Vite chunk-size warning. |

The Vite warning is not a functional failure, but it should remain visible for a later bundle-splitting pass.

## Browser Demo Results

Default local ports were already occupied on the workstation, so the clean-clone browser smoke first used isolated ports:

```bash
AGENT_HARBOR_DEMO_API_PORT=19094 \
AGENT_HARBOR_DEMO_FRONTEND_PORT=15184 \
MOCK_MCP_PORT=18794 \
VITE_API_BASE=http://127.0.0.1:19094 \
AGENT_HARBOR_CORS_ORIGINS=http://127.0.0.1:15184 \
  make demo
```

Browser result:

- Opened `http://127.0.0.1:15184/`.
- Landed on `#getting-started`.
- Showed the first-run setup chain.
- Used live API data, not sample fallback data.
- Did not show old Chinese product wording such as `上线证据` or `验收证据`.

This uncovered a public-readiness gap: isolated demo ports worked, but only if the evaluator already knew to set `VITE_API_BASE` and `AGENT_HARBOR_CORS_ORIGINS`.

After the fix in this branch, the same browser path was re-run with only the three documented port variables:

```bash
AGENT_HARBOR_DEMO_API_PORT=19095 \
AGENT_HARBOR_DEMO_FRONTEND_PORT=15185 \
MOCK_MCP_PORT=18795 \
  make demo
```

Browser result:

- Opened `http://127.0.0.1:15185/`.
- Landed on `http://127.0.0.1:15185/#getting-started`.
- Displayed `本地开发管理员`, making the local unauthenticated development mode explicit.
- Displayed `已读取真实接入数据`.
- Did not show sample data fallback.
- Did not show `上线证据` or `验收证据`.

Stopping the demo with `Ctrl+C` returned `Error 130`, which is the expected interrupt path for `make demo`.

## Fixes Applied

- `scripts/demo.sh` now derives `VITE_API_BASE` from the selected API port when the caller does not set it explicitly.
- `scripts/demo.sh` now derives `AGENT_HARBOR_CORS_ORIGINS` from the selected frontend port when the caller does not set it explicitly.
- README now documents the isolated-port `make demo` command without hidden frontend variables.
- `docs/engineering/release-checklist.md` now includes the same isolated-port browser check.
- The local development audit actor is now labeled `Local dev administrator` / `本地开发管理员`.
- `frontend/tests/productDocs.test.mjs` now guards the demo script behavior so port overrides do not regress.
- `frontend/tests/i18n.test.mjs` and `frontend/tests/permissionFlowLayout.test.mjs` now guard the updated local development actor label.

## Current Branch Verification

After applying the fixes, the audit branch was verified with:

| Check | Result | Notes |
| --- | --- | --- |
| `bash -n scripts/demo.sh` | Pass | Demo script syntax is valid. |
| `pnpm --dir frontend exec node --test tests/productDocs.test.mjs tests/i18n.test.mjs tests/permissionFlowLayout.test.mjs` | Pass | `57` focused tests passed. |
| `pnpm --dir frontend test` | Pass | `206` frontend tests passed. |
| `pnpm --dir frontend build` | Pass | Build completed with the existing Vite chunk-size warning. |
| `make check` | Pass | Go checks, frontend tests/build, script syntax, and GitHub YAML checks passed. |
| `make release-check` | Pass | Uncached Go tests, production hardening, frontend tests/build, script syntax, and GitHub YAML checks passed. |
| Isolated-port browser smoke | Pass | With only `AGENT_HARBOR_DEMO_API_PORT=19095`, `AGENT_HARBOR_DEMO_FRONTEND_PORT=15185`, and `MOCK_MCP_PORT=18795`, the console landed on `#getting-started` and loaded live data. |

## Post-Audit Follow-Up Status

- The frontend production build chunk-size warning has been addressed after the audit. Vite now splits stable vendor dependencies and lazy-loaded workspace panels into dedicated production chunks. The latest local build emitted `index` at `484.29 kB`, `react-vendor` at `189.63 kB`, `AiAdminPermissionWorkbench` at `38.63 kB`, `TenantOrganizationView` at `22.29 kB`, `CapabilityGovernanceView` at `15.23 kB`, and `icons-vendor` at `10.05 kB`, with no chunk-size warning.
- Dependabot PRs #72 and #73 were merged after this audit and `main` release gates passed after those merges.

## Remaining Non-Blocking Follow-Ups

- Historical engineering review documents still mention older wording for traceability. Product UI, README product path, product docs, and release checklist do not contain the Chinese word `证据`.

## Release Recommendation

Do not tag v0.2.0 public preview until this audit PR is merged and `main` CI is green again.

After merge, the release owner should run:

```bash
git pull --ff-only
make check
make release-check
AGENT_HARBOR_DEMO_API_PORT=19094 \
AGENT_HARBOR_DEMO_FRONTEND_PORT=15184 \
MOCK_MCP_PORT=18794 \
  make demo
```

Then open `http://127.0.0.1:15184/` and confirm the first screen lands on **Getting Started** with live data.
