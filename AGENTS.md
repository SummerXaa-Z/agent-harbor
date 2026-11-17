# AgentHarbor Agent Instructions

Read `CONTRIBUTING.md` and `SECURITY.md` before making changes. They remain the source of truth for contribution, verification, and private vulnerability reporting. Use the toolchain pins and existing Makefile targets rather than inventing parallel workflows.

## Engineering boundaries

- Keep changes focused and inspect the current implementation before choosing an approach. Prefer clear APIs, types, and narrow executable checks over growing prose rules for recurring mistakes.
- Preserve tenant/workspace scoping, deny precedence, capability approvals, credential redaction, and transactional audit guarantees. Test both permitted and rejected paths; a successful request alone does not validate an authorization boundary.
- Exercise security-sensitive changes with synthetic identities and disposable data. Never use production secrets or customer data as fixtures. PostgreSQL integration tests recreate the public schema: use only a dedicated test database.
- Verify gateway and upstream boundaries separately. Application tests do not prove deployment network isolation; follow `SECURITY.md` and report any unverified deployment assumptions.

## Evidence before completion

- Run `make check` as required by `CONTRIBUTING.md`; use `make release-check`, PostgreSQL integration, and relevant scenario targets for the changes described there. A PostgreSQL skip notice is not an integration-test pass.
- Match validation to the changed behavior: replay protocol inputs through the actual handler, read persisted results back, and exercise changed UI controls and repeated/error flows. State clearly when a test uses mocks or an isolated component rather than the running application.
- For a bug fix, reproduce the failure before the fix where feasible. For a new verification gate, demonstrate that it detects a controlled representative fault, then restore the correct code and rerun it. Do not commit the injected fault.
- Screenshots support visual review; repeatable screenshots alone do not prove correct content or working interactions. Do not substitute test counts, build success, or an agent's assertion for acceptance evidence.
- Do not disable tests, soften failures, broaden allowlists, or change expected results merely to obtain green CI. Legitimate changes to gates or security boundaries need explicit rationale and independent review of the resulting coverage.
- Before an authorized merge, review the final diff and checks for the exact final head. Revalidate after edits or conflict resolution, and verify the remote merge result. Authorization to edit is not authorization to merge, deploy, or release.
- Report what changed, what actually passed, and what failed or was not run. If an environment blocks validation, give the specific blocker instead of claiming readiness.

## Keep text purposeful

- Explain non-obvious reasons, invariants, compatibility constraints, and security boundaries in comments; do not narrate obvious code or preserve the agent's conversation and implementation diary.
- Update the existing documentation required by `CONTRIBUTING.md` when behavior changes. Avoid unrequested duplicate plans, reports, summaries, or guides; put concise verification evidence in the PR.
- Do not add teaching or implementation commentary to the product UI. Preserve necessary operational guidance, error messages, accessibility text, API documentation, and license notices.
- Review only the current diff for redundant comments, documentation, and UI copy. Do not turn a focused task into unrelated cleanup.
