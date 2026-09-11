package mysql

import "testing"

func TestMutationValuesUsesAllowlistedColumns(t *testing.T) {
	columns, args, err := mutationValues(
		[]string{"name", "idOs"},
		[]string{"name", "idOs"},
		map[string]any{"name": "chrome", "idOs": 1, "ignored": "must-not-persist"},
	)
	if err != nil {
		t.Fatalf("mutation values failed: %v", err)
	}
	if len(columns) != 2 || columns[0] != "name" || columns[1] != "idOs" {
		t.Fatalf("unexpected columns: %#v", columns)
	}
	if len(args) != 2 || args[0] != "chrome" || args[1] != 1 {
		t.Fatalf("unexpected values: %#v", args)
	}
}

func TestMutationValuesRejectsMissingRequiredField(t *testing.T) {
	if _, _, err := mutationValues([]string{"brand"}, []string{"brand"}, map[string]any{}); err == nil {
		t.Fatal("expected missing required field error")
	}
}
