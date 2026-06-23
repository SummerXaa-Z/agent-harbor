package security

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestParseCredentialKeyRejectsRepeatedRawKey(t *testing.T) {
	_, err := ParseCredentialKey("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err == nil {
		t.Fatalf("expected repeated raw credential key to be rejected")
	}
}

func TestParseCredentialKeyRejectsRepeatedPatternRawKey(t *testing.T) {
	_, err := ParseCredentialKey("0123456789abcdef0123456789abcdef")
	if err == nil {
		t.Fatalf("expected repeated pattern raw credential key to be rejected")
	}
}

func TestParseCredentialKeyRejectsZeroBase64Key(t *testing.T) {
	_, err := ParseCredentialKey(base64.StdEncoding.EncodeToString(make([]byte, CredentialKeySize)))
	if err == nil {
		t.Fatalf("expected zero base64 credential key to be rejected")
	}
}

func TestParseCredentialKeyAcceptsHighDiversityBase64Key(t *testing.T) {
	raw := []byte("AgentHarborCredentialKey-2026!!!")
	if len(raw) != CredentialKeySize {
		t.Fatalf("test key must be %d bytes, got %d", CredentialKeySize, len(raw))
	}
	key, err := ParseCredentialKey(base64.StdEncoding.EncodeToString(raw))
	if err != nil {
		t.Fatalf("parse credential key: %v", err)
	}
	if got := string(key); got != string(raw) {
		t.Fatalf("parsed key mismatch: %q", got)
	}
}

func TestParseCredentialKeyRejectsCommonWeakRawKey(t *testing.T) {
	_, err := ParseCredentialKey(strings.Repeat("password", 4))
	if err == nil {
		t.Fatalf("expected common weak raw credential key to be rejected")
	}
}
