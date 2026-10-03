package security

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
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

func testCredentialKeyA(t *testing.T) []byte {
	key := []byte("AgentHarborCredentialKey-2026!!!")
	if len(key) != CredentialKeySize {
		t.Fatalf("test key must be %d bytes, got %d", CredentialKeySize, len(key))
	}
	return key
}

func testCredentialKeyB(t *testing.T) []byte {
	key := make([]byte, CredentialKeySize)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return key
}

func TestEncryptDecryptCredentialsRoundTrip(t *testing.T) {
	credentials := map[string]string{
		"api_token": "secret-token-value-123",
		"password":  "hunter2-but-longer",
	}
	blob, err := EncryptCredentials(credentials, testCredentialKeyA(t))
	if err != nil {
		t.Fatalf("encrypt credentials: %v", err)
	}
	if len(blob) == 0 {
		t.Fatalf("expected a ciphertext blob")
	}
	if bytes.Contains(blob, []byte("secret-token-value-123")) {
		t.Fatalf("ciphertext embeds the plaintext credential value")
	}
	decrypted, err := DecryptCredentials(blob, testCredentialKeyA(t))
	if err != nil {
		t.Fatalf("decrypt credentials: %v", err)
	}
	if len(decrypted) != len(credentials) {
		t.Fatalf("round trip lost entries: %v", decrypted)
	}
	for name, value := range credentials {
		if decrypted[name] != value {
			t.Fatalf("round trip mismatch for %q: %q", name, decrypted[name])
		}
	}
}

func TestEncryptDecryptCredentialsEmptyEdgeCases(t *testing.T) {
	blob, err := EncryptCredentials(map[string]string{}, testCredentialKeyA(t))
	if err != nil {
		t.Fatalf("encrypt empty credentials: %v", err)
	}
	if len(blob) != 0 {
		t.Fatalf("expected empty credentials to encrypt to no blob, got %d bytes", len(blob))
	}
	decrypted, err := DecryptCredentials(nil, testCredentialKeyA(t))
	if err != nil {
		t.Fatalf("decrypt empty blob: %v", err)
	}
	if len(decrypted) != 0 {
		t.Fatalf("expected empty credentials from empty blob, got %v", decrypted)
	}
}

func TestEncryptCredentialsProducesUniqueCiphertexts(t *testing.T) {
	credentials := map[string]string{"api_token": "secret-token-value-123"}
	first, err := EncryptCredentials(credentials, testCredentialKeyA(t))
	if err != nil {
		t.Fatalf("encrypt first: %v", err)
	}
	second, err := EncryptCredentials(credentials, testCredentialKeyA(t))
	if err != nil {
		t.Fatalf("encrypt second: %v", err)
	}
	if bytes.Equal(first, second) {
		t.Fatalf("expected fresh nonces to produce distinct ciphertexts")
	}
}

func TestDecryptCredentialsRejectsWrongKey(t *testing.T) {
	blob, err := EncryptCredentials(map[string]string{"api_token": "secret"}, testCredentialKeyA(t))
	if err != nil {
		t.Fatalf("encrypt credentials: %v", err)
	}
	if _, err := DecryptCredentials(blob, testCredentialKeyB(t)); err == nil {
		t.Fatalf("expected decryption with a different key to fail")
	}
}

func TestDecryptCredentialsRejectsTamperedCiphertext(t *testing.T) {
	blob, err := EncryptCredentials(map[string]string{"api_token": "secret"}, testCredentialKeyA(t))
	if err != nil {
		t.Fatalf("encrypt credentials: %v", err)
	}
	for _, index := range []int{len(blob) / 2, len(blob) - 1} {
		tampered := append([]byte(nil), blob...)
		tampered[index] ^= 0xff
		if _, err := DecryptCredentials(tampered, testCredentialKeyA(t)); err == nil {
			t.Fatalf("expected tampered ciphertext at byte %d to fail authentication", index)
		}
	}
}

func TestDecryptCredentialsRejectsTruncatedBlob(t *testing.T) {
	key := testCredentialKeyA(t)
	if _, err := DecryptCredentials([]byte{1, 2, 3}, key); err == nil {
		t.Fatalf("expected a blob shorter than the nonce to be rejected")
	}
	// Empty blobs short-circuit to "no credentials" before key validation, so
	// the wrong-size key checks need a non-empty payload to reach credentialGCM.
	if _, err := DecryptCredentials([]byte{1, 2, 3}, key[:CredentialKeySize-1]); err == nil {
		t.Fatalf("expected a wrong-size key to be rejected on decrypt")
	}
	if _, err := EncryptCredentials(map[string]string{"api_token": "secret"}, key[:CredentialKeySize-1]); err == nil {
		t.Fatalf("expected a wrong-size key to be rejected on encrypt")
	}
}

func TestDecryptCredentialsRejectsAuthenticatedNonJSONPlaintext(t *testing.T) {
	key := testCredentialKeyA(t)
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("test cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("test gcm: %v", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		t.Fatalf("test nonce: %v", err)
	}
	blob := append(nonce, gcm.Seal(nil, nonce, []byte("authenticated but not json"), nil)...)
	if _, err := DecryptCredentials(blob, key); err == nil {
		t.Fatalf("expected authenticated non-JSON plaintext to be rejected at unmarshal")
	}
}
