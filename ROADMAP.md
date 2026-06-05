# Roadmap

AgentHarbor is focused on tenant-first authorization and evidence for AI agent, MCP, OpenAPI, and governed data access.

This roadmap is intentionally high level. Detailed implementation work should still go through issues and pull requests.

## Near Term

- Add explain-oriented management MCP tools for access decisions, failed simulations, and data-scope conflicts.
- Persist permission package versions and application records for repeatable review.
- Harden tenant access profiles with clearer inherited-vs-direct permission views.
- Expand MCP capability governance with richer tool metadata, approval history, and stale-capability handling.
- Improve local contributor ergonomics with clearer setup checks and seeded scenario data.
- Add production deployment examples for PostgreSQL-backed local and containerized environments.

## Next

- Add approval policy for high-risk permission package application.
- Add OpenAPI capability discovery and assignment semantics alongside MCP tools.
- Add first-class data-system targets for data lakes, warehouses, and databases.
- Expand data-scope validation so administrators can catch invalid narrowing before runtime.
- Improve audit exports and trace filtering for security review workflows.

## Later

- Add identity-provider integration for management-console operators.
- Add policy simulation before publishing tenant, workspace, or caller-instance changes.
- Add observability integrations for metrics, traces, and structured audit sinks.
- Define versioned API compatibility guarantees after the first tagged release.

## Non-Goals For The First Public Release

- Replacing a full IAM system.
- Granting unrestricted access to private-network upstream targets.
- Inferring every possible tool argument schema without explicit capability metadata.
- Supporting production multi-region deployment before the core permission model is stable.
