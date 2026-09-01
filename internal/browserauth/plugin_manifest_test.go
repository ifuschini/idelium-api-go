package browserauth
import "testing"
func TestValidPluginCodeRequiresObject(t *testing.T){if validPluginCode(nil)||validPluginCode([]any{"x"}){t.Fatal("non-object manifest accepted")};if !validPluginCode(map[string]any{"name":"demo"}){t.Fatal("object manifest rejected")}}
