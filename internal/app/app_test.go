package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostgresCredentialKeyFromEnvRequiresKey(t *testing.T) {
	t.Setenv("AGENT_HARBOR_CREDENTIAL_KEY", "")

	_, err := postgresCredentialKeyFromEnv()
	if err == nil {
		t.Fatalf("expected PostgreSQL credential key config error")
	}
}

func TestPostgresCredentialKeyFromEnvParsesRawKey(t *testing.T) {
	t.Setenv("AGENT_HARBOR_CREDENTIAL_KEY", "0123456789abcdef0123456789abcdef")

	key, err := postgresCredentialKeyFromEnv()
	if err != nil {
		t.Fatalf("parse credential key: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(key))
	}
}

func TestNewAllowsPrivateUpstreamsFromEnv(t *testing.T) {
	t.Setenv("AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS", "true")

	app, err := New(context.Background())
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer app.Close()

	body := map[string]any{
		"name":        "Local MCP Target",
		"workspaceId": "ws-local",
		"channelType": "mcp",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": "http://127.0.0.1:8099/mcp",
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("private upstream env flag should allow local endpoint, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPrivateUpstreamsEnvRejectsInvalidBoolean(t *testing.T) {
	t.Setenv("AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS", "sometimes")

	_, err := privateUpstreamsAllowedFromEnv()
	if err == nil {
		t.Fatalf("expected invalid private upstream env value to fail")
	}
}
