# Wrapper audit synchronization (M2 PR-2)

`harbor-wrap` enforces local policy and writes JSONL evidence without a network
connection. `harbor sync` imports that evidence into the shared audit timeline.
Imported records are operator-reported evidence, not independently witnessed
gateway authorization decisions.

## Setup and usage

Build the two commands:

```sh
go build -o ./bin/harbor ./cmd/harbor
go build -o ./bin/harbor-wrap ./cmd/harbor-wrap
```

Run the server with normal administrator authentication. Use PostgreSQL for
durable server storage; the default in-memory store is for bounded local
evaluation and loses its state on restart.

Register both caller and target through `POST /api/v1/agents` with
`channelType: local`, without endpoints or credentials. Use their returned IDs
in the policy and the wrapper's `--agent` and `--target` options. `harbor import`
remains HTTP-only; automated stdio wiring belongs to PR-3.

```sh
./bin/harbor-wrap --agent <registered-caller-id> --target <registered-target-id> \
  --policy ./harbor.yaml --spool ./audit-spool -- <your-stdio-server-command>

# Set HARBOR_ADMIN_KEY to an existing platform/tenant operator key, not an Agent Key.
# HARBOR_URL defaults to http://127.0.0.1:9090. Use HTTPS remotely.
./bin/harbor sync --spool ./audit-spool
./bin/harbor audit --json
./bin/harbor why <trace-id>
```

`--url` and `--admin-key` are also supported. Prefer the environment for the key
to avoid shell history, and never commit credentials. Redirects are rejected.
Sync does not edit client configuration, issue credentials, or pull policy
snapshots.

## Authentication and ingestion contract

`POST /api/v1/audit/traces/ingest` requires a platform or tenant administrator.
Security reviewers cannot import. Both registered identities must be inside the
operator's management scope. A workspace-restricted operator requires both
identities to have that concrete workspace. Unknown identities are rejected.
Disabled identities may supply historical evidence; importing does not enable
them or authorize new calls.

The body is `{ "events": [<wrap record>, ...] }`, with 1–100 events and the API's
1 MiB request limit. A successful response is HTTP 200 with
`{ "code": 0, "data": { "accepted": N } }`. The entire batch is validated before
writing. Identical event IDs with identical evidence are idempotent. A changed
record under an existing ID returns HTTP 409 (`TRACE_ID_CONFLICT`) and rolls back
the entire batch. An import cannot adopt an existing gateway trace's ID.

Only `source: wrap` and `routeKey: tools/call` records are accepted. Tenant and
workspace are derived server-side from the registered caller, not uploaded
scope fields. Plaintext arguments and gateway grant evidence are not accepted.
The wrapper's historical `capabilityId` field contains the raw MCP tool name;
ingestion stores this as `toolName`, leaving registry `capabilityId` empty.
Allowed calls require outcome `ok` or `error`. Denied calls accept `denied` or a
legacy empty outcome, normalized to `denied`. Existing gateway records display
source `gateway`; an empty historical outcome means unknown, not success.

Migration 018 adds source, outcome, tool name, and private ingest receipts.
PostgreSQL inserts the batch atomically and retains original timestamp
nanoseconds in the receipt for exact retry comparison.

## Reliability and retention boundaries

- The wrapper writes unique, owner-only `wrap-*.jsonl` files and records argument
  digests, not plaintext arguments.
- Sync snapshots each file's current length. It defers an incomplete final line
  until a later run. A malformed complete line stops progress rather than being
  silently skipped.
- Uploads contain at most 100 events and 512 KiB; a single line is limited to
  64 KiB. This keeps uploads below the API request cap.
- A batch must be acknowledged before its checkpoint is atomically replaced and
  fsynced. Checkpoints are namespaced by destination and a hash of the admin key.
  A lost acknowledgement or checkpoint causes replay, handled idempotently.
- Sync never truncates or deletes source spool files. Concurrent sync runs can
  regress a checkpoint and repeat uploads, but do not discard evidence.
- Use `--replay` after a server reset or when deliberately reimporting retained
  evidence. Between-run changes to an already-read prefix cause replay. Do not
  externally mutate an active spool file. Archive closed files only after
  verifying persistence and keeping suitable backups.
- The memory store bounds visible trace history but keeps volatile receipts for
  unique imported IDs. It is only suitable for bounded evaluation, not durable
  operation or an unlimited ingest stream. Use PostgreSQL for durable receipts;
  SQLite support is future work.
- Wrapper spool errors are reported to stderr while tool forwarding continues.
  There is no quota/rotation/drop ledger or per-event fsync in this milestone.
  Full disks, power loss, or process death can lose local events. This is not a
  universal losslessness guarantee.

Exit codes: `0` success, `2` local configuration/format/checkpoint failure, `3`
scope denial, and `4` transport/remote failure. Sync reports checkpoint progress
so a partial run can be retried without guessing which records reached the
server.

## Verification

```sh
go test -race ./cmd/harbor ./internal/httpapi ./internal/store ./internal/wrap
make check
make release-check
```

The integration test exercises wrapper output, an offline attempt, recovery,
checkpoint reuse, duplicate replay, scoped audit lookup, and `why`. PostgreSQL
integration coverage additionally verifies durable atomic/idempotent ingestion.
Use registered caller and target IDs when running the workflow manually.
