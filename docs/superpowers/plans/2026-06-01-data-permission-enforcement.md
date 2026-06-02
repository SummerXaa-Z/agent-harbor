# Data Permission Enforcement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Enforce tenant/workspace/instance data permissions in the existing MCP capability governance path by validating scope narrowing at assignment time, computing effective scopes at runtime, injecting a trusted runtime context header for upstream agents/tools, and recording the effective scopes in trace evidence.

**Architecture:** Keep Agent Harbor as the permission control plane. Data scopes remain structured metadata in `domain.DataScope`; Agent Harbor validates hierarchy and forwards a signed-by-topology context envelope, while downstream MCP servers/agents apply concrete data predicates. This avoids rewriting arbitrary tool arguments before we have per-tool schema semantics.

**Tech Stack:** Go standard library, current in-memory and Postgres stores, existing `internal/httpapi` handlers, existing Jest/Vitest frontend unchanged unless tests reveal a necessary UI correction.

---

## Task 1: Add Domain Scope Narrowing Helpers

- [x] Add exported helpers in `internal/domain/data_scope.go`.

Implementation shape:

```go
func EffectiveDataScopes(parent []DataScope, child []DataScope) ([]DataScope, bool) {
    if len(parent) == 0 {
        return CloneDataScopes(child), true
    }
    if len(child) == 0 {
        return CloneDataScopes(parent), true
    }

    effective := make([]DataScope, 0, len(child))
    for _, childScope := range child {
        matched := false
        for _, parentScope := range parent {
            scope, ok := mergeDataScope(parentScope, childScope)
            if ok {
                effective = append(effective, scope)
                matched = true
                break
            }
        }
        if !matched {
            return nil, false
        }
    }
    return effective, true
}
```

Rules:
- A scope list is an OR-list.
- Every child scope must be equal to or narrower than at least one parent alternative.
- Empty parent scope list means no established boundary yet, so the child becomes the effective boundary.
- Empty child scope list inherits the parent effective boundary.
- A child can fill an empty parent dimension, but cannot change a non-empty parent dimension.

- [x] Add `internal/domain/data_scope_test.go`.

Required cases:
- Equal scope is accepted.
- Empty child inherits parent.
- Empty parent accepts child.
- Child can fill an empty parent field.
- Child cannot change a fixed parent field.
- Multiple alternatives accept if one parent alternative contains the child.
- A child list with one unmatched alternative rejects the whole list.

## Task 2: Use Effective Scopes in Store Evaluation

- [x] Update `internal/store/memory.go` `EvaluateCapabilityAccess` to derive scopes step by step:

```go
tenantScopes, ok := domain.EffectiveDataScopes(capability.DataScopes, tenantEntitlement.DataScopes)
workspaceScopes, ok := domain.EffectiveDataScopes(tenantScopes, workspaceAssignment.DataScopes)
instanceScopes, ok := domain.EffectiveDataScopes(workspaceScopes, instanceAssignment.DataScopes)
```

Fail closed when persisted state contains an invalid expansion.

- [x] Mirror the same runtime calculation in `internal/store/postgres.go`.

- [x] Extend memory store tests with inherited and narrowed scopes.

Expected effective scope example:

```go
[]domain.DataScope{{
    TenantID: "tenant-a",
    WorkspaceID: "workspace-a",
    Region: "us-east",
}}
```

- [x] Extend Postgres round-trip test where practical, preserving the existing DB-env skip behavior.

## Task 3: Validate Scope Narrowing in Control-Plane Handlers

- [x] Add small HTTP handler helpers in `internal/httpapi/server.go`:

```go
func validateDataScopesNarrow(parent []domain.DataScope, child []domain.DataScope) ([]domain.DataScope, bool) {
    return domain.EffectiveDataScopes(parent, child)
}
```

Use capability scopes as the parent for tenant entitlements, effective tenant scopes as the parent for workspace assignments, and effective workspace scopes as the parent for instance assignments.

- [x] Update handlers:
- `createTenantEntitlement`
- `createWorkspaceAssignment`
- `createInstanceAssignment`

Required behavior:
- Reject widening with `400 Bad Request`.
- Omitted child `dataScopes` is allowed and inherits the parent.
- Existing not-found and duplicate behavior stays unchanged.

- [x] Add server tests:
- Tenant entitlement rejecting a scope outside capability.
- Workspace assignment rejecting a scope outside tenant entitlement.
- Instance assignment with omitted scopes inherits workspace/tenant/capability scopes in the runtime decision.

## Task 4: Inject Trusted Runtime Context for MCP Tool Calls

- [x] Add `internal/httpapi/context_header.go` with:

```go
const agentHarborContextHeader = "X-AgentHarbor-Context"

type agentHarborContextPayload struct {
    SchemaVersion string             `json:"schemaVersion"`
    TenantID      string             `json:"tenantId"`
    WorkspaceID   string             `json:"workspaceId"`
    TargetID      string             `json:"targetId"`
    CallerSubject string             `json:"callerSubject"`
    CapabilityID  string             `json:"capabilityId"`
    CapabilityKey string             `json:"capabilityKey"`
    ToolName      string             `json:"toolName"`
    DataScopes    []domain.DataScope `json:"dataScopes"`
}
```

Encode as base64url JSON with `base64.RawURLEncoding`.

- [x] Treat `X-AgentHarbor-Context` as reserved:
- Do not copy caller-provided values from inbound requests.
- Do not forward configured static or credential header values for this name.
- Set the generated header after all other header copy/config/credential logic in governed MCP `tools/call`.

- [x] Update `proxyUpstreamIfConfigured` to accept an optional request mutator:

```go
type upstreamRequestMutator func(*http.Request) error
```

Use it only for governed MCP tool calls, so normal route proxying remains unchanged.

- [x] Add server test with an upstream HTTP server:
- Caller sends a spoofed `X-AgentHarbor-Context`.
- Target config also tries to set the same header.
- Upstream receives exactly the Agent Harbor generated payload.
- Decoded payload includes effective `dataScopes`.

## Task 5: Document and Demo the New Enforcement Layer

- [x] Update `README.md` with concise data-permission semantics:
- Scope lists are OR alternatives.
- Child scopes must be equal/narrower than parent scopes.
- Runtime calls forward `X-AgentHarbor-Context`.
- Agent Harbor does not yet rewrite arbitrary tool arguments.

- [x] Add a sprint demo script if the existing demo pattern is still current:
- `scripts/demo-sprint13-data-permission-enforcement.sh`
- Include it in `scripts/demo-all.sh` and `Makefile` only if those files already follow the same pattern cleanly.

## Task 6: Verify

- [x] Run focused tests first:

```sh
go test ./internal/domain ./internal/store ./internal/httpapi
```

- [x] Run full Go tests if focused tests pass:

```sh
go test ./...
```

- [x] Run frontend tests only if frontend files changed:

```sh
npm test -- --runInBand
```

- [x] Inspect `git diff --check` and `git status --short`.

## Task 7: Finish

- [x] Commit the implementation on `codex/data-permission-enforcement`.
- [x] Push or prepare PR depending on remote availability.
- [x] Summarize behavior, tests, and any remaining limitations.
