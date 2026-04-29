package attachments_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/and1truong/liveboard/internal/attachments"
)

func TestStorePutThenOpen(t *testing.T) {
	dir := t.TempDir()
	s := attachments.NewStore(dir)
	payload := []byte("hello world")
	desc, err := s.Put(bytes.NewReader(payload), "greeting.txt")
	if err != nil {
		t.Fatalf("put: %v", err)
	}

	wantHash := sha256.Sum256(payload)
	wantHex := hex.EncodeToString(wantHash[:])
	if desc.Hash[:64] != wantHex {
		t.Errorf("hash prefix got %q want %q", desc.Hash[:64], wantHex)
	}
	if desc.Size != int64(len(payload)) {
		t.Errorf("size got %d want %d", desc.Size, len(payload))
	}
	if desc.Mime == "" {
		t.Errorf("mime should be sniffed, got empty")
	}

	r, err := s.Open(desc.Hash)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = r.Close() }()
	got, _ := io.ReadAll(r)
	if !bytes.Equal(got, payload) {
		t.Errorf("read mismatch")
	}

	if _, err := os.Stat(filepath.Join(dir, ".attachments", desc.Hash)); err != nil {
		t.Errorf("pool file missing: %v", err)
	}
}

func TestStorePutDedupes(t *testing.T) {
	dir := t.TempDir()
	s := attachments.NewStore(dir)
	a, _ := s.Put(bytes.NewReader([]byte("x")), "a.txt")
	b, _ := s.Put(bytes.NewReader([]byte("x")), "b.txt")
	if a.Hash != b.Hash {
		t.Errorf("expected dedup: %q vs %q", a.Hash, b.Hash)
	}
}

func TestStoreRemove(t *testing.T) {
	dir := t.TempDir()
	s := attachments.NewStore(dir)
	d, _ := s.Put(bytes.NewReader([]byte("y")), "x.txt")
	if err := s.Remove(d.Hash); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := s.Open(d.Hash); err == nil {
		t.Errorf("expected open to fail after remove")
	}
}
