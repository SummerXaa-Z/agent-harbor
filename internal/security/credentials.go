package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const CredentialKeySize = 32

func ParseCredentialKey(raw string) ([]byte, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil && len(decoded) == CredentialKeySize {
		return decoded, nil
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(trimmed); err == nil && len(decoded) == CredentialKeySize {
		return decoded, nil
	}
	if len([]byte(trimmed)) == CredentialKeySize {
		return []byte(trimmed), nil
	}
	return nil, fmt.Errorf("credential key must be 32 bytes raw or base64-encoded")
}

func EncryptCredentials(credentials map[string]string, key []byte) ([]byte, error) {
	if len(credentials) == 0 {
		return nil, nil
	}
	gcm, err := credentialGCM(key)
	if err != nil {
		return nil, err
	}
	plaintext, err := json.Marshal(credentials)
	if err != nil {
		return nil, fmt.Errorf("marshal credentials: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("credential nonce: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ciphertext...), nil
}

func DecryptCredentials(blob []byte, key []byte) (map[string]string, error) {
	if len(blob) == 0 {
		return map[string]string{}, nil
	}
	gcm, err := credentialGCM(key)
	if err != nil {
		return nil, err
	}
	if len(blob) <= gcm.NonceSize() {
		return nil, fmt.Errorf("credential ciphertext is invalid")
	}
	nonce := blob[:gcm.NonceSize()]
	ciphertext := blob[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt credentials: %w", err)
	}
	var credentials map[string]string
	if err := json.Unmarshal(plaintext, &credentials); err != nil {
		return nil, fmt.Errorf("unmarshal credentials: %w", err)
	}
	if credentials == nil {
		credentials = map[string]string{}
	}
	return credentials, nil
}

func credentialGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != CredentialKeySize {
		return nil, fmt.Errorf("credential key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("credential cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("credential gcm: %w", err)
	}
	return gcm, nil
}
