# Sprint 9 Route Policies Plan

## Phase 1: Failing Tests

- Add HTTP tests for route policy create/list/update/disable.
- Add data-plane tests proving allow and deny policies affect MCP route decisions.
- Add PostgreSQL integration coverage for route policy round trip and policy-vs-grant precedence.

## Phase 2: Domain and Store

- Add route policy request/response/domain types.
- Extend repository interface with route policy CRUD and route access evaluation.
- Implement memory store policy ranking and legacy grant fallback.
- Add PostgreSQL migration and store implementation.

## Phase 3: HTTP API

- Register admin routes.
- Validate caller/target existence, effect, status, priority, route type, and route key.
- Record management audit events for create/update/disable.
- Use repository route access decision in the data plane.

## Phase 4: Frontend and Demo

- Add route policy API client and types.
- Load route policies directly, falling back to sample data only when offline.
- Make the Route Governance form create policies instead of legacy grants.
- Add a Sprint 9 demo script covering allow, deny, update, disable, and trace evidence.

## Phase 5: Verification

- Run Go unit tests.
- Run frontend build.
- Run the Sprint 9 demo script against a local server.
- Run static checks and request review.
