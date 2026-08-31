package integrations

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestLaravelSecretEnvelopeRoundTripAndRedaction(t *testing.T) {
	keyValue := "base64:" + base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	key, err := ParseApplicationKey(keyValue)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := EncryptLaravelString(key, "super-secret-value")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(envelope, "super-secret-value") {
		t.Fatal("ciphertext exposed plaintext")
	}
	plaintext, err := DecryptLaravelString(key, envelope)
	if err != nil || plaintext != "super-secret-value" {
		t.Fatalf("unexpected decrypted value or error: %q %v", plaintext, err)
	}
	if _, err := DecryptLaravelString(key, envelope+"tampered"); err == nil || strings.Contains(err.Error(), "super-secret-value") {
		t.Fatalf("expected safe ciphertext validation, got %v", err)
	}
}

func TestApplicationKeyValidationIsSafe(t *testing.T) {
	for _, value := range []string{"", "base64:not-valid", "short"} {
		if _, err := ParseApplicationKey(value); err != ErrInvalidApplicationKey {
			t.Fatalf("expected safe invalid-key error for %q, got %v", value, err)
		}
	}
}

func TestDecryptsLaravelGeneratedCiphertextFixture(t *testing.T) {
	key, err := ParseApplicationKey("base64:MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=")
	if err != nil {
		t.Fatal(err)
	}
	fixture := "eyJpdiI6IjdLM1pOdnJMQUQwNDlDZEtDWUZEQkE9PSIsInZhbHVlIjoiSFQxeWR6dlpFcHZsbGxndFBOV3ZuQmUrOVBQdkNUYWgrdmwycjZZVENXRT0iLCJtYWMiOiJlYmZhN2UxYzhjMjczNWU5ZTJlYmY3Mjg4MTNjYTcwNTJjNjFiNjI2NmEzNTU3ZjU5M2FlYjAzMTE0Y2MzYzNmIiwidGFnIjoiIn0="
	plaintext, err := DecryptLaravelString(key, fixture)
	if err != nil || plaintext != "laravel-compatible-secret" {
		t.Fatalf("failed to decrypt Laravel fixture: %q %v", plaintext, err)
	}
}
