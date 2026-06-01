# Contributing

AgentHarbor is a clean-room implementation track. Keep changes small, reviewable, and independently verifiable.

## Clean-Room Boundary

Do not copy source code, migrations, tests, deployment scripts, adapter code, generated assets, styles, or component structure from the existing Rust/React runtime. Use only product requirements, public protocols, and public product references.

When a change depends on product behavior, record the requirement in docs, tests, or PR notes so reviewers can audit the source of the behavior without comparing against legacy implementation code.

Do not file public issues for suspected vulnerabilities, credential leaks, authorization bypasses, or audit integrity failures. Follow the private reporting path in `SECURITY.md`.

## Branches And PRs

- Start from the latest `main`.
- Use `codex/<short-topic>` for Codex-authored branches.
- Keep each PR focused on one behavior, workflow, or documentation boundary.
- Prefer a new small PR over a long-lived stack. If stacking is unavoidable, document the review order and retarget bottom-up.
- Keep PR descriptions current with scope, verification, and follow-ups.
- CODEOWNERS currently routes all review ownership to `@SummerXaa-Z`; split ownership by surface only when there are stable additional maintainers.
- Use GitHub issue forms for public bug reports and feature proposals; blank issues are disabled so reports keep scope, reproduction, and acceptance details.

## Local Verification

Run the standard local checks before opening or updating a PR:

```bash
make check
```

This runs backend tests, vet, build, frontend tests/build, and demo script syntax checks.
It also parse-checks GitHub YAML configuration via `make github-config-lint`.

Use uncached Go tests before behavior-sensitive review:

```bash
make release-check
```

`make release-check` runs the uncached Go test path plus vet, build, frontend test/build, and demo script syntax checks.

Run PostgreSQL integration when a change touches repository behavior, migrations, transactions, credentials, audit events, route policies, or CI database wiring:

```bash
AGENT_HARBOR_TEST_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable' \
  make test-postgres
```

With a local API already running, use the full demo suite for end-to-end smoke coverage:

```bash
make demo-all
```

If admin protection is enabled on the server, pass the same key to demo scripts:

```bash
ADMIN_KEY=local-admin-key make demo-all
```

## CI Expectations

GitHub Actions runs the same Makefile targets used locally:

- Backend: `make test`, `make vet`, `make build`, `make demo-scripts-lint`
- GitHub configuration lint: `make github-config-lint`
- PostgreSQL integration: `make test-postgres`
- Frontend: `make frontend-test`, `make frontend-build`

CI runs on pull requests and on pushes to `main`. It uses one concurrency group per workflow and branch, so new commits cancel superseded runs on the same branch. Reviewers should rely on the latest run for the branch or PR.

CI jobs have explicit timeout budgets: 10 minutes for Backend, 15 minutes for PostgreSQL integration, and 10 minutes for Frontend. Treat a timeout as a workflow defect or unexpectedly slow test path, not as a flaky pass.

Do not mark work complete until the relevant local checks and GitHub checks have both passed.

For larger behavior changes, downstream handoffs, or tagged releases, use the release checklist in `docs/engineering/release-checklist.md`.

## Dependency Updates

Dependabot opens weekly update PRs for Go modules, frontend npm packages, and GitHub Actions. Review and merge them using `docs/engineering/dependency-updates.md`.

## Documentation

Update docs alongside behavior changes:

- `README.md` for user-facing runtime, API, and local development entrypoints.
- `docs/sprints/` for sprint-level scope and acceptance notes.
- `docs/engineering/` for process, review, or governance workflow.
- `CHANGELOG.md` for session-level decisions, verification, and lessons learned.

Keep examples executable. Prefer `make` targets and scripts over long command sequences that drift from CI.
