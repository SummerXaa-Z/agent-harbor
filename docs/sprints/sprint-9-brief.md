# Sprint 9 Brief: Route Policy Objects

Status: Implementing on `codex/sprint-9-route-policies`

## Goal

Turn Route Governance from a UI projection of access grants into first-class route policy objects that can allow or deny agent-to-agent routes with explicit priority.

## User Stories

- As a platform owner, I can create, list, update, and disable route policies independently from legacy access grants.
- As a runtime operator, an enabled deny policy can block a matching route even when a legacy grant still exists.
- As a developer, existing access-grant based demos keep working while new route policies become the preferred control-plane primitive.

## Acceptance Criteria

- `POST /api/v1/route-policies`, `GET /api/v1/route-policies`, `PATCH /api/v1/route-policies/{id}`, and `DELETE /api/v1/route-policies/{id}` are available behind the admin gate.
- Route policies include scope, caller, target, route type/key, effect, status, priority, and timestamps.
- Route policy creation rejects caller/target pairs across tenant or workspace boundaries.
- Data-plane authorization evaluates enabled route policies before falling back to legacy access grants.
- Higher-priority policies win; deny wins ties; disabled policies are ignored.
- Policy mutations append management audit events without leaking secrets.
- PostgreSQL and memory repositories share the same repository contract and behavior.
- The console reads route policies from the API when available and still falls back to sample data when offline.

## Non-goals

- No route-level retry overrides yet.
- No expression language, ABAC, or group-based policy subjects.
- No transactional outbox for policy audit events.
- No migration that rewrites existing access grants into route policies.
