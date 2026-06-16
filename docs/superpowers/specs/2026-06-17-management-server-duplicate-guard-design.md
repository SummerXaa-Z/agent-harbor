# Management Server Duplicate Guard Design

## Problem

Resource Management now disables submit buttons while frontend mutations are running, but production callers can still retry the same REST or management-client request outside the browser. Repeated create or rotate calls can create duplicate Agents, duplicate route policies, multiple short-lived Agent keys, or unnecessary credential-rotation audit events.

## Decision

Add semantic duplicate guards on the server instead of a generic idempotency table.

Generic idempotency keys are not the right first step because `POST /api/v1/agent-keys` returns one-time plaintext key material. Replaying that response would require persisting sensitive plaintext or returning an inconsistent replay response. The safer production slice is to reject obvious duplicate resource mutations with `409 DUPLICATE_RESOURCE_MUTATION`, while making identical credential rotations a no-op that returns the current Agent without appending audit noise.

## Scope

- Agent creation: reject another non-disabled Agent in the same tenant/workspace with the same normalized name and channel type.
- Agent key creation: reject another active, unrevoked key for the same Agent and key name created in the recent duplicate window.
- Credential rotation: if the submitted credential map is identical to the current credential map, return the current Agent unchanged and do not append a new audit event.
- Route policy creation: reject another non-disabled policy with the same caller, target, route type, route key, effect, priority, and retry settings.
- Frontend: map `DUPLICATE_RESOURCE_MUTATION` to bilingual operator copy in Resource Management forms.

## Non-Goals

- No generic `Idempotency-Key` persistence table.
- No replay of Agent key plaintext.
- No new database migration.
- No changes to permission package apply semantics.

## Verification

- HTTP API tests cover duplicate Agent, duplicate Agent key, identical credential rotation no-op, and duplicate route policy.
- Frontend i18n and management hook structure tests cover the new duplicate mutation copy.
- Full `make check` and `make release-check` remain green.
