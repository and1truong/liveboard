package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/and1truong/liveboard/internal/attachments"
)

// inlineMimes is the small allowlist of MIMEs served with
// Content-Disposition: inline. Everything else is served as attachment so
// the browser cannot execute it in our origin (XSS hardening).
var inlineMimes = map[string]struct{}{
	"image/png":       {},
	"image/jpeg":      {},
	"image/gif":       {},
	"image/webp":      {},
	"application/pdf": {},
}

// postAttachment handles POST /api/v1/attachments (multipart, single file
// part named "file"). Returns the stored Descriptor as JSON.
func (d Deps) postAttachment(w http.ResponseWriter, r *http.Request) {
	maxBytes := d.AttachmentMaxBytes
	if maxBytes <= 0 {
		maxBytes = 25 << 20
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	if err := r.ParseMultipartForm(maxBytes); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
			return
		}
		writeError(w, fmt.Errorf("%w: %v", errInvalid, err))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, fmt.Errorf("%w: missing file part: %v", errInvalid, err))
		return
	}
	defer func() { _ = file.Close() }()

	desc, err := d.Attachments.Put(file, header.Filename)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(desc)
}

// getAttachment handles GET /api/v1/attachments/{hash}/{name}.
// Serves bytes with sniffed Content-Type, conservative Content-Disposition,
// long immutable cache, X-Content-Type-Options: nosniff.
func (d Deps) getAttachment(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	name := chi.URLParam(r, "name")

	f, err := d.Attachments.Open(hash)
	if err != nil {
		if errors.Is(err, attachments.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		writeError(w, err)
		return
	}
	defer func() { _ = f.Close() }()

	size, mime, err := d.Attachments.Stat(hash)
	if err != nil {
		writeError(w, err)
		return
	}

	// Optional thumbnail.
	if r.URL.Query().Get("thumb") == "1" {
		thumb, terr := attachments.Thumb(f, 256)
		if terr != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = io.Copy(w, thumb)
		return
	}

	disposition := "attachment"
	if _, ok := inlineMimes[mime]; ok {
		disposition = "inline"
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`%s; filename="%s"; filename*=UTF-8''%s`,
			disposition, asciiSafe(name), url.PathEscape(name)))

	if r.Method == http.MethodHead {
		return
	}
	_, _ = io.Copy(w, f)
}

// asciiSafe returns name with non-ASCII chars stripped, for the legacy
// `filename=` field. The RFC 5987 `filename*=` carries the real value.
func asciiSafe(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 32 && r < 127 && r != '"' && r != '\\' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "download"
	}
	return b.String()
}
