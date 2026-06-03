# Release Checklist

Use this checklist before merging larger behavior changes, cutting a tagged release, or declaring `main` ready for downstream integration.

## Required Local Checks

Run the uncached local release gate:

```bash
make release-check
```

Use the checked-in Go, Node, and pnpm pins when running release checks locally or in CI.

This covers:

- Go formatting via `make gofmt-check`
- uncached Go tests via `make test-fresh`
- `go vet` via `make vet`
- Go package build via `make build`
- frontend unit tests via `make frontend-test`
- frontend production build via `make frontend-build`
- scenario script syntax checks via `make scenario-scripts-lint`
- GitHub YAML configuration parse checks via `make github-config-lint`

Run PostgreSQL integration when the change touches persistence, migrations, audit behavior, credential storage, route policies, or CI database wiring:

```bash
AGENT_HARBOR_TEST_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable' \
  make test-postgres
```

Run end-to-end scenarios when user-visible control-plane or data-plane behavior changes. Start the API first, then run:

```bash
ADMIN_KEY=local-admin-key make scenario-all
```

For web console changes, run `make demo` and manually verify that the Cockpit Core Journey Workbench preflight checks pass, the journey can run against the local mock MCP server, and the non-destructive reset returns the browser session to the default demo scope. Confirm the core tenant access journey remains usable in both English and Simplified Chinese, including the language toggle, Tenant Permission Console, capability governance entry point, runtime evidence metrics, and trace evidence labels.

## GitHub Checks

Before merging:

- Confirm the PR CI is green for Backend, PostgreSQL integration, and Frontend.
- Confirm no job reached its timeout budget; timeouts indicate a workflow defect or unexpectedly slow test path that needs investigation.
- Confirm the PR description lists the local verification commands that were run.
- Confirm docs and scenarios describe behavior that exists in the current diff.

After merging:

- Confirm the `main` push CI is green.
- Pull the latest `main` locally.
- Check `git status --short --branch` is clean and tracking `origin/main`.

## Release Notes

For a tagged release or downstream handoff, summarize:

- behavior changes and compatibility notes
- schema or migration changes
- security and credential handling changes
- verification evidence, including PostgreSQL and scenario coverage when relevant
- known follow-ups that should not block the release

Keep release notes short, factual, and tied to merged commits or PRs.
