# Demo First-Run Experience Design

## Goal

Make AgentHarbor's first open-source browser evaluation usable from one local command and make the Core Journey Workbench explain what is missing before the user runs it.

## Context

The current core journey is production-representative once the API, mock MCP server, and frontend are running. The weak point is startup friction: a first-time evaluator must run three terminals and infer from failures whether the API, mock MCP, or private upstream flag is missing.

There is no tenant-level cascade delete API today. Demo data is isolated by fresh `ui-core-*` run IDs, so reset should not attempt destructive backend cleanup in this iteration.

## Design Consensus

### Local Demo Command

Add a dependency-free local demo runner under `scripts/` and expose it as `make demo`.

The runner starts:

- AgentHarbor API with `AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true`
- mock MCP server at `http://127.0.0.1:8787/mcp`
- frontend dev server at `http://127.0.0.1:5174`

The script prints the console URL and shuts down all child processes on `SIGINT` or `SIGTERM`. It should fail fast when required tools (`go`, `python3`, `pnpm`) are missing. It should also detect occupied ports and print the exact conflicting port before starting child processes.

This command is for local evaluation only. It must not imply production deployment.

### Console Preflight

Add a focused frontend module for first-run checks. The Core Journey Workbench should show three checks:

- API health: `GET /healthz` at the configured API base
- mock MCP health: `GET http://127.0.0.1:8787/healthz`
- local upstream readiness: a lightweight capability refresh probe is not safe before an MCP target exists, so the UI should infer this check from API health plus a clear warning that local MCP calls require `AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true`

When API or mock MCP is down, the workbench should show a clear Chinese/English message and disable the core journey run button until the missing service is available. If both are up, the button is enabled.

### Demo Reset

Add a non-destructive reset for the current browser session:

- clear the active core journey config and result
- reset filters and scope back to the default demo values
- refresh console data

Do not delete backend data yet. A backend reset would require a separate design for tenant subtree deletion, audit semantics, credential cleanup, and PostgreSQL referential behavior.

### Documentation

Update README so the primary browser evaluation path is:

```bash
make demo
```

Keep the three-terminal path as a troubleshooting/manual path. Document that demo data is isolated by run ID and that reset is UI/session reset, not destructive cleanup.

## Alternatives Considered

### Makefile-Only Process Orchestration

Rejected. Make can start commands, but robust signal handling and child cleanup are clearer in a script.

### Docker Compose

Deferred. It helps deployment-style demos, but it adds a second runtime path and is not necessary for the local developer-preview experience.

### Backend Demo Reset Endpoint

Deferred. It would make reset cleaner, but deleting a tenant subtree would touch agents, credentials, capabilities, entitlements, assignments, traces, and audit events. That is product behavior, not a small demo convenience.

## Testing Strategy

- Unit test preflight state derivation in the frontend.
- Shell syntax-check the demo runner through existing script linting.
- Run `make demo` manually, open the console, verify preflight checks pass, run the core journey to `6/6`, and verify reset clears the current session state without deleting historical data.
- Run `make release-check`.

## Open Decisions

None. The scope is intentionally limited to local first-run evaluation and non-destructive UI reset.
