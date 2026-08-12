# AgentHarbor repository guidance

## Product intent

- AgentHarbor is a tenant-first access-governance platform for agents, MCP tools, OpenAPI services, and governed data access.
- Preserve the core authorization chain: tenant -> workspace -> caller instance -> capability -> effective data scope. Never broaden scope or bypass approval/audit behavior unintentionally.
- This repository is developer preview software. Do not describe it as production-ready without fresh, relevant release validation.

## Local development and verification

- Follow the pinned toolchains in `go.mod`, `.node-version`, and `frontend/package.json`.
- Use `make demo` for first browser evaluation and `make check` for the normal local backend/frontend/static validation path.
- Use `make release-check` for release handoff, merge-critical, security-sensitive, or cross-journey changes. Do not run it by default for a narrow documentation edit unless its value is clear.
- Local loopback/private-upstream flags and unauthenticated-admin mode are development-only. Never carry them into deployment guidance or production configuration.

## Security and data handling

- Never add real admin keys, session secrets, tokens, identities, tenant data, or captured traces to the repository, tests, screenshots, or documentation.
- Preserve secure defaults: authenticated management access, bounded tenant/workspace visibility, approval requirements for risky changes, and auditability.
- When modifying API, permission, or UI behavior, explain the user journey and the authorization consequence in plain language.
