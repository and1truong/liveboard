// Package attachments implements the workspace-wide content-addressed blob
// pool plus reference scanning, garbage collection, and thumbnail generation.
package attachments

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// PoolDir is the per-workspace subdirectory holding all blobs.
const PoolDir = ".attachments"

// Descriptor describes a stored blob. JSON keys match the short-form
// Attachment shape used in card metadata and the TS Attachment type so an
// upload response can be passed straight into a subsequent add_attachments
// mutation without renaming.
type Descriptor struct {
	Hash string `json:"h"` // <sha256-hex>.<ext>  (ext derived from origName, may be empty)
	Name string `json:"n"` // origName, used as default download filename
	Size int64  `json:"s"`
	Mime string `json:"m"` // sniffed at Put time via http.DetectContentType
}

// ErrNotFound is returned when a hash isn't present in the pool.
var ErrNotFound = errors.New("attachment not found")

// Store is a content-addressed blob pool rooted at workspaceDir/<PoolDir>.
type Store struct {
	dir string // pool dir, e.g. /workspace/.attachments
}

// NewStore returns a Store rooted at workspaceDir. The pool dir is created
// lazily on first Put. Construction never fails.
func NewStore(workspaceDir string) *Store {
	return &Store{dir: filepath.Join(workspaceDir, PoolDir)}
}

// Dir returns the absolute path to the pool directory.
func (s *Store) Dir() string { return s.dir }

// Put streams r through a sha256 hasher and a size counter, materializing
// the blob in the pool. Dedup is automatic: a duplicate Put returns the
// existing descriptor without rewriting the file.
//
// origName is used to derive the on-disk file extension and is returned
// untouched in Descriptor.Name.
func (s *Store) Put(r io.Reader, origName string) (Descriptor, error) {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return Descriptor{}, fmt.Errorf("mkdir pool: %w", err)
	}
	tmp, err := os.CreateTemp(s.dir, ".upload-*")
	if err != nil {
		return Descriptor{}, fmt.Errorf("temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() // best-effort if anything below fails

	hasher := sha256.New()
	sizer := &countingWriter{}
	mw := io.MultiWriter(tmp, hasher, sizer)
	// Sniff first 512 bytes for MIME via tee.
	var sniffBuf [512]byte
	headLen, _ := io.ReadFull(r, sniffBuf[:])
	if headLen > 0 {
		if _, err := mw.Write(sniffBuf[:headLen]); err != nil {
			_ = tmp.Close()
			return Descriptor{}, err
		}
	}
	if _, err := io.Copy(mw, r); err != nil {
		_ = tmp.Close()
		return Descriptor{}, err
	}
	if err := tmp.Close(); err != nil {
		return Descriptor{}, err
	}

	ext := strings.ToLower(path.Ext(origName))
	hexsum := hex.EncodeToString(hasher.Sum(nil))
	hash := hexsum + ext
	dst := filepath.Join(s.dir, hash)

	// Dedup: if dst exists, drop tmp; otherwise rename.
	if _, statErr := os.Stat(dst); statErr != nil {
		// not present yet — move tmp into place
		if err := os.Rename(tmpPath, dst); err != nil {
			return Descriptor{}, fmt.Errorf("rename: %w", err)
		}
	}
	// else: already present; tmp will be removed by deferred cleanup

	mime := http.DetectContentType(sniffBuf[:headLen])
	return Descriptor{
		Hash: hash,
		Name: origName,
		Size: sizer.n,
		Mime: mime,
	}, nil
}

// Open returns a reader for hash. Caller must Close.
func (s *Store) Open(hash string) (*os.File, error) {
	if !validHash(hash) {
		return nil, fmt.Errorf("%w: invalid hash %q", ErrNotFound, hash)
	}
	f, err := os.Open(filepath.Join(s.dir, hash))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

// Stat returns size and mime for hash without opening the data stream.
// The mime is re-sniffed (cheap) since it isn't stored alongside.
func (s *Store) Stat(hash string) (size int64, mime string, err error) {
	if !validHash(hash) {
		return 0, "", fmt.Errorf("%w: invalid hash %q", ErrNotFound, hash)
	}
	fi, err := os.Stat(filepath.Join(s.dir, hash))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, "", ErrNotFound
		}
		return 0, "", err
	}
	f, err := os.Open(filepath.Join(s.dir, hash))
	if err != nil {
		return 0, "", err
	}
	defer func() { _ = f.Close() }()
	var head [512]byte
	n, _ := io.ReadFull(f, head[:])
	return fi.Size(), http.DetectContentType(head[:n]), nil
}

// Remove deletes hash from the pool. Idempotent: missing → nil.
func (s *Store) Remove(hash string) error {
	if !validHash(hash) {
		return fmt.Errorf("%w: invalid hash %q", ErrNotFound, hash)
	}
	err := os.Remove(filepath.Join(s.dir, hash))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// validHash sanity-checks a pool key. It must be exactly 64 lowercase hex
// chars (sha256), optionally followed by a "." plus a short alnum extension.
// Anything else is rejected to prevent path traversal and weird filenames.
func validHash(s string) bool {
	if len(s) < 64 {
		return false
	}
	for i := 0; i < 64; i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	if len(s) == 64 {
		return true
	}
	if s[64] != '.' {
		return false
	}
	for _, c := range s[65:] {
		switch {
		case c >= 'a' && c <= 'z':
		case c >= '0' && c <= '9':
		default:
			return false
		}
	}
	return len(s)-65 <= 16 // sane extension cap
}

type countingWriter struct{ n int64 }

func (c *countingWriter) Write(p []byte) (int, error) {
	c.n += int64(len(p))
	return len(p), nil
}
