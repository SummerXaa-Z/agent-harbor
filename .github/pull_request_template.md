## Summary

Describe the change in 2-4 bullets. Name the user-visible behavior, engineering contract, or documentation outcome.

## Scope

- [ ] This PR is one reviewable change.
- [ ] This PR is an integration checkpoint and the review order is documented below.
- [ ] This PR intentionally leaves follow-up work listed in the follow-up section.

## Review Boundary

Recommended review order:

1. Domain and API contracts
2. Store behavior and migrations
3. HTTP/data-plane behavior
4. Frontend surfaces
5. Docs, scenarios, and changelog

Focus areas:

Name the highest-risk files, behaviors, or compatibility boundaries reviewers should inspect first.

## Verification

- [ ] `make check`
- [ ] `make release-check`, when preparing larger behavior changes or release handoffs
- [ ] `git diff --check`
- [ ] PostgreSQL integration, when store or migration behavior changes
- [ ] Scenario script, when user-facing flows change

## Data And Security

- [ ] No plaintext credentials, tokens, or Agent Keys are returned by management APIs or audit metadata.
- [ ] PostgreSQL migrations are forward-only and safe for existing local data.
- [ ] Management endpoints preserve admin-key expectations.
- [ ] Data-plane authorization behavior is covered when route, grant, or policy semantics change.

## Follow-Ups

List known follow-ups that are deliberately outside this PR. Write `None` when there are no follow-ups.
