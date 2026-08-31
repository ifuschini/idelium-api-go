package auditlog

import (
	"reflect"
	"testing"
)

func TestRedactCopiesNestedSensitiveValues(t *testing.T) {
	input := map[string]any{"name": "demo", "apiKey": "unsafe", "nested": map[string]any{"password": "unsafe", "items": []any{map[string]any{"session_token": "unsafe", "status": "ok"}}}}
	want := map[string]any{"name": "demo", "apiKey": "[REDACTED]", "nested": map[string]any{"password": "[REDACTED]", "items": []any{map[string]any{"session_token": "[REDACTED]", "status": "ok"}}}}
	if got := Redact(input); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected redacted audit value: %#v", got)
	}
	if input["apiKey"] != "unsafe" {
		t.Fatal("redaction mutated the caller payload")
	}
}
