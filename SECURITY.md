# Security Policy

AgentHarbor handles authorization decisions, Agent Keys, upstream credentials, route policies, audit evidence, and data-scope enforcement. Please report security issues privately so fixes can be coordinated before details are disclosed.

## Supported Versions

Until tagged releases are published, security fixes target the latest `main` branch. After the first release, the supported versions table will be updated here.

| Version | Supported |
| --- | --- |
| `main` | Yes |

## Reporting a Vulnerability

Do not open a public issue for suspected vulnerabilities, credential leaks, authorization bypasses, or audit integrity failures.

Use GitHub private vulnerability reporting if it is enabled for this repository. If it is not enabled, contact the repository owner privately and share only the minimum information needed to establish a secure reporting channel.

Please include:

- affected commit, branch, tag, or deployment version
- impact summary
- reproduction steps or proof-of-concept payloads
- whether credentials, Agent Keys, tokens, database contents, tenant data, or audit records were exposed
- any known public disclosure deadline or external coordination requirement

Do not include live secrets, production credentials, private customer data, or unrelated logs. Redact tokens and keys before sharing evidence.

## Security Scope

Security reports should focus on confidentiality, integrity, availability, or auditability risks, including:

- management API access without the configured `X-Admin-Key` protection
- Agent Key acceptance outside its intended tenant, workspace, status, or TTL boundaries
- route policy, access grant, tenant entitlement, or deny-rule bypasses
- capability approval or assignment bypasses for MCP tools
- data-scope widening or enforcement bypasses
- plaintext credential exposure in API responses, traces, logs, scenario output, audit metadata, or persisted PostgreSQL rows
- credential encryption, rotation, or versioning regressions
- unaudited management mutations where transactional audit writes are expected
- proxy behavior that forwards secret headers incorrectly or leaks upstream credentials
- dependency, CI, or release process changes that weaken build, test, or supply-chain integrity

General bugs, documentation gaps, and local setup issues can use normal issues or pull requests unless they expose one of the risks above.

## Fix Handling

Security fixes should stay narrow and include verification evidence. At minimum, run:

```bash
make release-check
git diff --check
```

Run PostgreSQL integration when the issue touches persistence, transactions, credentials, migrations, audit events, tenant scope resolution, route policies, or capability assignments:

```bash
AGENT_HARBOR_TEST_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable' \
  make test-postgres
```

If the vulnerability affects a user-visible control-plane or data-plane flow, add or update a focused test and run the relevant scenario script before disclosure.

## Disclosure

Coordinate disclosure timing with the repository maintainers. A disclosure note should describe impact, affected versions or commits, the fix, verification evidence, and any required operator action.

Do not publish exploit details before a fix is merged and downstream users have had a reasonable chance to update.
