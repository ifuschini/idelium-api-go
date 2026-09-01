package browserauth

import "testing"

func TestEnvironmentHasInlineSecret(t *testing.T) {
	if environmentHasInlineSecret(map[string]any{"baseUrl": "https://example.test"}) {
		t.Fatal("ordinary configuration rejected")
	}
	if !environmentHasInlineSecret(map[string]any{"nested": []any{map[string]any{"password": "value"}}}) {
		t.Fatal("nested password was not rejected")
	}
	if !environmentHasInlineSecret(map[string]any{"API_TOKEN": "value"}) {
		t.Fatal("token key was not rejected")
	}
}
