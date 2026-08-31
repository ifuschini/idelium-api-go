// Package auditlog defines append-only audit data safety rules shared by all domains.
package auditlog

import "strings"

// Redact recursively replaces values stored under sensitive keys. It returns a
// copy and never mutates the caller's event payload.
func Redact(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		redacted := make(map[string]any, len(typed))
		for key, child := range typed {
			if SensitiveKey(key) {
				redacted[key] = "[REDACTED]"
			} else {
				redacted[key] = Redact(child)
			}
		}
		return redacted
	case []any:
		redacted := make([]any, len(typed))
		for index, child := range typed {
			redacted[index] = Redact(child)
		}
		return redacted
	default:
		return typed
	}
}

func SensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(key, "_", ""))
	for _, marker := range []string{"password", "passwd", "secret", "token", "apikey", "authorization", "cookie", "credential", "session", "recovery"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}
