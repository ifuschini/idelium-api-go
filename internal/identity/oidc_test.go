package identity

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestValidateOIDCJWTClaimsAndSignature(t *testing.T) {
	t.Setenv("IDELIUM_OIDC_ISSUER", "https://issuer.example")
	t.Setenv("IDELIUM_OIDC_AUDIENCE", "idelium-api")
	t.Setenv("IDELIUM_OIDC_HS256_SECRET", "test-signing-secret")
	token := testOIDCToken(t, map[string]any{
		"iss": "https://issuer.example", "sub": "user-1", "aud": "idelium-api",
		"exp": time.Now().Add(time.Minute).Unix(), "nonce": "nonce-1", "email": "user@example.org",
	})
	claims, err := validateOIDCJWT(token, "nonce-1")
	if err != nil || claims.Subject != "user-1" {
		t.Fatalf("validateOIDCJWT() = %#v, %v", claims, err)
	}
}

func TestValidateOIDCJWTRejectsIssuerAudienceNonceAndSignature(t *testing.T) {
	t.Setenv("IDELIUM_OIDC_ISSUER", "https://issuer.example")
	t.Setenv("IDELIUM_OIDC_AUDIENCE", "idelium-api")
	t.Setenv("IDELIUM_OIDC_HS256_SECRET", "test-signing-secret")
	base := map[string]any{"iss": "https://issuer.example", "sub": "user-1", "aud": "idelium-api", "exp": time.Now().Add(time.Minute).Unix(), "nonce": "nonce-1"}
	for name, mutate := range map[string]func(map[string]any){
		"issuer":   func(c map[string]any) { c["iss"] = "https://other.example" },
		"audience": func(c map[string]any) { c["aud"] = "other-api" },
		"nonce":    func(c map[string]any) { c["nonce"] = "other-nonce" },
	} {
		t.Run(name, func(t *testing.T) {
			claims := map[string]any{}
			for k, v := range base {
				claims[k] = v
			}
			mutate(claims)
			if _, err := validateOIDCJWT(testOIDCToken(t, claims), "nonce-1"); err == nil {
				t.Fatal("expected claim validation failure")
			}
		})
	}
	parts := strings.Split(testOIDCToken(t, base), ".")
	parts[2] = base64.RawURLEncoding.EncodeToString([]byte("invalid"))
	if _, err := validateOIDCJWT(strings.Join(parts, "."), "nonce-1"); err == nil {
		t.Fatal("expected signature validation failure")
	}
}

func testOIDCToken(t *testing.T, claims map[string]any) string {
	t.Helper()
	enc := func(v any) string {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	header, payload := enc(map[string]string{"alg": "HS256", "typ": "JWT"}), enc(claims)
	mac := hmac.New(sha256.New, []byte(os.Getenv("IDELIUM_OIDC_HS256_SECRET")))
	_, _ = mac.Write([]byte(header + "." + payload))
	return header + "." + payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
