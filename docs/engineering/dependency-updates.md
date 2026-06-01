# Dependency Updates

AgentHarbor uses Dependabot to open weekly dependency update PRs for:

- Go modules in `/`
- frontend npm packages in `/frontend`
- GitHub Actions in `.github/workflows`

Dependabot runs on Monday morning in the `Asia/Shanghai` timezone and limits each ecosystem to five open PRs.

Routine minor and patch updates are grouped per ecosystem:

- `go-minor-patch` for Go modules
- `frontend-minor-patch` for frontend npm packages
- `actions-minor-patch` for GitHub Actions

Major updates stay outside these groups so they can be reviewed as planned compatibility work.

## Review Policy

Treat dependency PRs like normal code changes:

- Read the changed package, version, and changelog when the impact is not obvious.
- Prefer one ecosystem update PR at a time unless the updates are clearly coupled or already grouped by Dependabot.
- Keep major-version updates separate from routine patch/minor updates.
- Do not merge dependency PRs that change generated lockfiles or manifests without green CI.

## Verification

For routine patch/minor updates:

```bash
make check
git diff --check
```

For major updates or dependency changes that can affect runtime behavior:

```bash
make release-check
```

Run PostgreSQL integration when Go dependency updates touch database drivers, transaction behavior, migrations, or test infrastructure:

```bash
AGENT_HARBOR_TEST_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable' \
  make test-postgres
```

Run `make demo-all` against a local API when updates touch HTTP routing, frontend runtime behavior, or demo tooling.

## Merge Rule

Merge dependency PRs only after:

- PR CI is green.
- The PR description or review comment records which local checks were run.
- `main` is still current enough that the dependency PR is not hiding conflicts with recent infrastructure changes.
