# Sprint 4 Design: Secret Header Injection

## Scope

Sprint 4 adds create-time upstream credentials for target Agents. Credentials are submitted outside `channelConfig`, redacted from all management responses, persisted encrypted in PostgreSQL, and injected into upstream proxy requests through a non-secret mapping.

## API Shape

Create Agent accepts:

```json
{
  "name": "Credentialed MCP",
  "workspaceId": "workspace-demo",
  "channelType": "mcp",
  "status": "active",
  "channelConfig": {
    "endpoint": "https://api.example.com/mcp",
    "credentialHeaders": {
      "Authorization": "apiToken"
    }
  },
  "credentials": {
    "apiToken": "Bearer ..."
  }
}
```

Management responses return the Agent but omit `credentials`. `credentialHeaders` is allowed to contain secret-bearing header names because it stores only a credential reference, not a value.

## Validation

- `credentials` keys are trimmed, non-empty, and unique after trimming.
- `credentials` values must be non-empty strings.
- `channelConfig.credentialHeaders` must be an object of string-to-string values.
- Each `credentialHeaders` value must reference an existing credential key.
- `channelConfig.headers` continues to reject secret-like header names.
- All outbound URL validation remains unchanged for `endpoint` and `specUrl`.

## Proxy Behavior

After route authorization and trace recording, proxy request construction runs in this order:

1. Copy safe caller headers such as `Content-Type` and `Accept`.
2. Copy target `channelConfig.headers` for non-secret upstream headers.
3. Copy target `channelConfig.credentialHeaders` by resolving each value from Agent `Credentials`.

Credential headers are only injected for allowed data-plane calls.

## Persistence

PostgreSQL migration `002_sprint4_agent_credentials.sql` adds `agents.credentials_ciphertext bytea`.

Non-empty credentials are encrypted with AES-GCM. The nonce is prepended to ciphertext. `AGENT_HARBOR_CREDENTIAL_KEY` accepts raw or base64 32-byte key material and is required when PostgreSQL persistence is enabled. Empty credentials remain empty ciphertext for compatibility.

Credential keys are short identifiers, not display names or secret material. They must be 1-64 characters, start with a letter, and then use letters, digits, `_`, `-`, or `.`.

## Follow-up

- Add credential rotation and partial update APIs.
- Add per-credential metadata such as last rotated time and masked display hints.
- Add AES-GCM AAD and key id/version fields before building rotation.
- Consider KMS envelope encryption when deployment topology hardens.
