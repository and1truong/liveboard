package attachments_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/and1truong/liveboard/internal/attachments"
)

func TestGCRemovesOrphans(t *testing.T) {
	dir := t.TempDir()
	s := attachments.NewStore(dir)
	keep, _ := s.Put(bytes.NewReader([]byte("keep")), "k.txt")
	orphan, _ := s.Put(bytes.NewReader([]byte("orphan")), "o.txt")

	board := "---\nversion: 1\nname: A\n---\n\n## C\n\n- [ ] x\n  attachments: [{\"h\":\"" + keep.Hash + "\",\"n\":\"k\",\"s\":4,\"m\":\"text/plain\"}]\n"
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte(board), 0o644); err != nil {
		t.Fatal(err)
	}

	deleted, err := attachments.GC(dir)
	if err != nil {
		t.Fatalf("gc: %v", err)
	}
	if len(deleted) != 1 || deleted[0] != orphan.Hash {
		t.Errorf("deleted = %v, want [%q]", deleted, orphan.Hash)
	}
	if _, err := s.Open(keep.Hash); err != nil {
		t.Errorf("kept blob gone: %v", err)
	}
	if _, err := s.Open(orphan.Hash); err == nil {
		t.Errorf("orphan still present")
	}
}

func TestGCNoPoolDirIsNoop(t *testing.T) {
	dir := t.TempDir()
	deleted, err := attachments.GC(dir)
	if err != nil {
		t.Fatalf("gc: %v", err)
	}
	if len(deleted) != 0 {
		t.Errorf("got deleted: %v", deleted)
	}
}
