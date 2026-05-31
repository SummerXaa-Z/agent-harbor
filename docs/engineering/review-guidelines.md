# Review Guidelines

AgentHarbor changes should be easy to review in small slices, even when the branch is a larger sprint stack. Use this guide with `.github/pull_request_template.md` when preparing or reviewing pull requests.

## Preferred PR Shape

- One PR should represent one behavior slice: a domain contract, a management API, a data-plane behavior, a frontend workflow, or a documentation/demo update.
- A stacked/integration PR is acceptable as a draft checkpoint, but the PR description must say so and list the review order.
- If a PR changes more than one runtime surface and the review order is not obvious, split it before marking it ready.

## Review Order For Sprint Stacks

1. Domain and API contracts: verify request/response fields, defaults, validation, and backward compatibility.
2. Store behavior and migrations: verify memory and PostgreSQL behavior match, migrations are forward-only, and integration tests cover persistence.
3. HTTP and data plane: verify admin-key behavior, authorization decisions, proxy semantics, trace/audit evidence, and error responses.
4. Frontend surfaces: verify forms reject invalid input, tables show the new state clearly, and API fallback data remains coherent.
5. Docs and demos: verify README, sprint brief, changelog, and demo scripts describe behavior that the code actually implements.

## Required Verification Evidence

Every ready PR should state which commands were run. Use the full set when runtime code changes:

```bash
make check
git diff --check
```

Run PostgreSQL integration when a PR changes store behavior, migrations, audit persistence, credentials, or route policy evaluation:

```bash
AGENT_HARBOR_TEST_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable' \
  make test-postgres
```

Use `make release-check` instead of `make check` before larger behavior merges or release handoffs so uncached Go tests are part of the evidence.

Run the relevant demo script when a user-facing workflow changes. Demo scripts should prove the public behavior and fail if expected evidence is missing.

## Security And Governance Checks

- Credential values, Agent Key plaintext, tokens, cookies, and authorization headers must not appear in management responses, audit metadata, traces, docs, or demo output.
- Management endpoints must preserve admin-key expectations when `AGENT_HARBOR_ADMIN_KEY` is set.
- Data-plane authorization changes must cover both allowed and denied paths.
- Audit changes must explain whether writes are best-effort, transactional, or intentionally out of scope.
- PostgreSQL migrations must be additive or clearly safe for existing local development data.

## When To Split

Split a PR before marking it ready when any of these are true:

- Reviewers need to understand frontend, HTTP, store, and migration changes at the same time to approve one behavior.
- The PR contains both infrastructure/process work and runtime behavior.
- Demo/docs updates describe behavior from more than one sprint.
- CI is green, but there is no narrow review path through the diff.

Draft PRs may be large when they act as integration checkpoints. Ready PRs should be boring to review.
