package cardid

import (
	"strings"
	"testing"
)

func TestNewIDLength(t *testing.T) {
	id := NewID()
	if len(id) != 10 {
		t.Fatalf("want len 10, got %d (%q)", len(id), id)
	}
}

func TestNewIDAlphabet(t *testing.T) {
	id := NewID()
	for _, r := range id {
		if !strings.ContainsRune(Alphabet, r) {
			t.Fatalf("rune %q not in alphabet", r)
		}
	}
}

func TestNewIDUnique(t *testing.T) {
	seen := make(map[string]struct{}, 10000)
	for i := 0; i < 10000; i++ {
		id := NewID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate id %q after %d draws", id, i)
		}
		seen[id] = struct{}{}
	}
}

func TestNewIDOverride(t *testing.T) {
	orig := NewID
	t.Cleanup(func() { NewID = orig })
	NewID = func() string { return "FIXED00001" }
	if got := NewID(); got != "FIXED00001" {
		t.Fatalf("override failed: %q", got)
	}
}
