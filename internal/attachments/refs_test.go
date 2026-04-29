package attachments_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/and1truong/liveboard/internal/attachments"
)

func TestCollectReferenced(t *testing.T) {
	dir := t.TempDir()
	hashA := strings.Repeat("a", 64) + ".pdf"
	hashB := strings.Repeat("b", 64) + ".png"
	hashC := strings.Repeat("c", 64) + ".txt"

	board1 := `---
version: 1
name: A
---

## C

- [ ] Card
  attachments: [{"h":"` + hashA + `","n":"x","s":0,"m":"x"}]
  Body has an image: ![](attachment:` + hashB + `)
`
	board2 := `---
version: 1
name: B
---

## C

- [ ] Card
  attachments: [{"h":"` + hashC + `","n":"y","s":0,"m":"x"}]
`
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte(board1), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "b.md"), []byte(board2), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := attachments.CollectReferenced(dir)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	for _, want := range []string{hashA, hashB, hashC} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing hash %q in %v", want, got)
		}
	}
}

func TestCollectReferencedSkipsPool(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".attachments"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A .md file inside the pool dir must NOT be scanned (would create
	// false self-references).
	hash := strings.Repeat("d", 64) + ".pdf"
	body := `---
version: 1
name: X
---
## C
- [ ] x
  attachments: [{"h":"` + hash + `","n":"x","s":0,"m":"x"}]
`
	if err := os.WriteFile(filepath.Join(dir, ".attachments", "ignore.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := attachments.CollectReferenced(dir)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if _, ok := got[hash]; ok {
		t.Errorf("hash from pool-dir .md leaked into refs: %v", got)
	}
}
