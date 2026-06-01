# Security Policy

AgentHarbor handles authorization decisions, Agent Keys, upstream credentials, route policies, and audit evidence. Treat security reports as private until a fix and disclosure path are agreed.

## Supported Versions

This clean-room implementation track only supports the latest `main` branch. Security fixes should target `main` first and may be backported only if a downstream integration branch explicitly requests it.

## Reporting A Vulnerability

Do not open a public GitHub issue for suspected vulnerabilities, credential leaks, or bypasses. Use GitHub private vulnerability reporting if it is available for this repository. If it is not available, contact the repository owner privately and share only the minimum information needed to establish a private reporting channel.

Include:

- affected commit, branch, or PR
- impact summary
- reproduction steps or proof-of-concept payloads
- whether credentials, Agent Keys, tokens, database contents, or audit records were exposed
- any public disclosure deadline or external coordination constraint

Do not include live secrets, production credentials, private customer data, or unrelated logs. Redact tokens and keys before sharing evidence.

## Security Scope

Report issues that can affect confidentiality, integrity, or auditability, including:

- management API access without the expected `X-Admin-Key` protection
- Agent Key acceptance outside its intended tenant, workspace, status, or TTL boundaries
- route policy, access grant, or deny-rule bypasses
- plaintext credential exposure in API responses, traces, logs, demo output, audit metadata, or persisted PostgreSQL rows
- credential encryption, rotation, or versioning regressions
- unaudited management mutations where the code expects transactional audit writes
- proxy behavior that forwards secret headers incorrectly or leaks upstream credentials
- dependency or CI changes that weaken build, test, or release integrity

General bugs, documentation gaps, and local setup issues can use normal pull requests or issues unless they expose one of the security risks above.

## Handling Fixes

Security fixes should stay narrow and include verification evidence. At minimum, run:

```bash
make release-check
git diff --check
```

Run PostgreSQL integration when the issue touches persistence, transactions, credentials, migrations, audit events, or route policy storage:

```bash
AGENT_HARBOR_TEST_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable' \
  make test-postgres
```

If the vulnerability affects a user-visible control-plane or data-plane flow, add or update a focused test and run the relevant demo script before disclosure.

## Disclosure

Coordinate disclosure timing with the repository owner. A disclosure note should describe impact, affected versions or commits, the fix, verification evidence, and any required operator action. Do not publish exploit details before the fix is merged and downstream consumers have had a reasonable chance to update.
