package app

import "testing"

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
