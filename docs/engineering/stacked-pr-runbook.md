# Stacked PR Runbook

This repository currently uses draft PRs to review the AgentHarbor sprint stack without rewriting the already-pushed branch history.

## Current Stack

| PR | Base | Head | Review Scope |
| --- | --- | --- | --- |
| #2 | `main` | `codex/sprint-2-governance-proxy` | Sprint 2 governance proxy cleanup |
| #3 | `codex/sprint-2-governance-proxy` | `codex/sprint-3-mcp-policy-controls` | Sprint 3 MCP method policy controls |
| #4 | `codex/sprint-3-mcp-policy-controls` | `codex/sprint-4-secret-header-injection` | Sprint 4 secret header injection |
| #5 | `codex/sprint-4-secret-header-injection` | `codex/sprint-5-proxy-retry-classification` | Sprint 5 proxy retry classification |
| #6 | `codex/sprint-5-proxy-retry-classification` | `codex/sprint-6-runtime-metrics` | Sprint 6 runtime metrics |
| #7 | `codex/sprint-6-runtime-metrics` | `codex/sprint-7-credential-rotation` | Sprint 7 credential rotation |
| #8 | `codex/sprint-7-credential-rotation` | `codex/sprint-8-management-audit` | Sprint 8 management audit |
| #9 | `codex/sprint-8-management-audit` | `codex/sprint-9-route-policies` | Sprint 9 route policy objects |
| #1 | `codex/sprint-9-route-policies` | `codex/sprint-10-route-policy-retry-overrides` | Sprint 10 retry, Sprint 11 transactional audit, CI, PR governance |

The branch ancestry is linear:

```text
main
  -> codex/sprint-2-governance-proxy
  -> codex/sprint-3-mcp-policy-controls
  -> codex/sprint-4-secret-header-injection
  -> codex/sprint-5-proxy-retry-classification
  -> codex/sprint-6-runtime-metrics
  -> codex/sprint-7-credential-rotation
  -> codex/sprint-8-management-audit
  -> codex/sprint-9-route-policies
  -> codex/sprint-10-route-policy-retry-overrides
```

## Ready Order

Review from bottom to top:

1. Mark #2 ready first after re-running its focused verification.
2. Merge or intentionally preserve #2 before marking #3 ready.
3. Continue upward through #9.
4. Keep #1 draft until the lower stack is reviewed or intentionally retained as an integration checkpoint.

Do not mark a higher PR ready while its base PR is still materially changing.

## CI Status Caveat

The CI workflow was introduced in the top branch, so the current lower draft PRs (#2-#9) do not have status checks yet. Treat #1 as the current full integration signal, and use local verification before marking lower PRs ready.

Before marking a lower PR ready, run the relevant focused checks on that PR's head branch, then paste the evidence into the PR body:

```bash
go test ./...
go vet ./...
go build ./...
pnpm --dir frontend test
pnpm --dir frontend build
git diff --check
```

Run PostgreSQL integration when the PR changes store behavior or migrations:

```bash
AGENT_HARBOR_TEST_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable' \
  go test ./internal/store -count=1
```

## Options For CI-First Review

If automated checks are required on every lower PR before review, choose one of these strategies deliberately:

- Merge a separate CI foundation PR first, then rebase or merge each sprint branch onto that foundation branch.
- Merge the current stack bottom-up into `main`, relying on #1 as the integration signal until CI reaches `main`.
- Keep #2-#9 as draft documentation PRs and review the fully checked top PR #1 as the temporary integration checkpoint.

Avoid force-pushing the stack unless reviewers agree on the new topology first.

## Merge Discipline

- Prefer bottom-up merges.
- After each merge, retarget the next PR to the updated base branch if GitHub does not do so automatically.
- Re-run CI on the next ready PR before merging it.
- Keep `CHANGELOG.md` entries with the sprint or process change that introduced them.
