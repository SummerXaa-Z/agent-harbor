# Contributing

AgentHarbor is a product codebase for tenant-first AI agent access governance. Keep changes small, reviewable, and independently verifiable.

## Source Boundary

Do not import proprietary implementation artifacts, generated assets, styles, component structures, migrations, tests, or deployment scripts from another codebase.

When a change depends on product behavior, record the requirement in docs, tests, or PR notes so reviewers can audit what the change is intended to provide.

Do not file public issues for suspected vulnerabilities, credential leaks, authorization bypasses, or audit integrity failures. Follow the private reporting path in `SECURITY.md`.

## Branches And PRs

- Start from the latest `main`.
- Use descriptive branch names such as `feature/<short-topic>`, `fix/<short-topic>`, or `docs/<short-topic>`.
- Keep each PR focused on one behavior, workflow, or documentation boundary.
- Prefer a new small PR over a long-lived stack. If stacking is unavoidable, document the review order and retarget bottom-up.
- Keep PR descriptions current with scope, verification, and follow-ups.
- CODEOWNERS currently routes all review ownership to `@SummerXaa-Z`; split ownership by surface only when there are stable additional maintainers.
- Use GitHub issue forms for public bug reports and feature proposals; blank issues are disabled so reports keep scope, reproduction, and acceptance details.

## Local Verification

Editors should honor the root `.editorconfig`: keep text files UTF-8 with LF endings and final newlines, use tabs for Go files and `Makefile`, and use two-space indentation elsewhere. Git also normalizes text files to LF via `.gitattributes`.

Use the checked-in toolchain pins for local verification: Go is declared in `go.mod`, Node is declared in `.node-version`, and the frontend pnpm version is declared in `frontend/package.json`.

Run the standard local checks before opening or updating a PR:

```bash
make check
```

This runs Go formatting checks, backend tests, vet, build, frontend tests/build, and scenario script syntax checks.
It also parse-checks GitHub YAML configuration via `make github-config-lint`.

Use `make fmt` before committing Go changes when `make gofmt-check` reports files that need formatting.

Use uncached Go tests before behavior-sensitive review:

```bash
make release-check
```

`make release-check` runs Go formatting checks, the uncached Go test path, vet, build, frontend test/build, and scenario script syntax checks.

Run PostgreSQL integration when a change touches repository behavior, migrations, transactions, credentials, audit events, route policies, or CI database wiring:

```bash
AGENT_HARBOR_TEST_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable' \
  make test-postgres
```

With a local API already running, use the full scenario suite for end-to-end smoke coverage:

```bash
make scenario-all
```

If admin protection is enabled on the server, pass the same key to scenario scripts:

```bash
ADMIN_KEY=local-admin-key make scenario-all
```

## CI Expectations

GitHub Actions runs the same Makefile targets used locally:

- Backend: `make gofmt-check`, `make test`, `make vet`, `make build`, `make scenario-scripts-lint`
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
- `docs/engineering/` for process, review, or governance workflow.
- `CHANGELOG.md` for public release notes and notable changes.

Keep examples executable. Prefer `make` targets and scripts over long command sequences that drift from CI.
