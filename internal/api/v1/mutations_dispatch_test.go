package v1_test

import (
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/and1truong/liveboard/internal/api/v1"
)

func TestDispatchAddCard(t *testing.T) {
	deps := newTestDeps(t)
	path := filepath.Join(deps.Workspace.Dir, "demo.md")

	op := v1.MutationOp{
		Type:    "add_card",
		AddCard: &v1.AddCardOp{Column: "Todo", Title: "dispatched"},
	}
	if err := v1.Dispatch(deps.Engine, path, -1, op); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	raw, _ := os.ReadFile(path)
	if !contains(string(raw), "dispatched") {
		t.Errorf("card not written: %s", raw)
	}
}

func TestDispatchUnknownOpErrors(t *testing.T) {
	deps := newTestDeps(t)
	path := filepath.Join(deps.Workspace.Dir, "demo.md")
	op := v1.MutationOp{Type: "bogus"}
	if err := v1.Dispatch(deps.Engine, path, -1, op); err == nil {
		t.Fatal("want error, got nil")
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (stringContains(haystack, needle))
}

func stringContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
