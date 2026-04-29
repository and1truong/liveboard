package v1_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	v1 "github.com/and1truong/liveboard/internal/api/v1"
	"github.com/and1truong/liveboard/internal/attachments"
)

// newAttachmentDeps returns a Deps with a fresh attachments.Store rooted in
// a tempdir, plus the optional AttachmentMaxBytes override.
func newAttachmentDeps(t *testing.T, maxBytes int64) v1.Deps {
	t.Helper()
	d := newTestDeps(t)
	d.Attachments = attachments.NewStore(d.Dir)
	d.AttachmentMaxBytes = maxBytes
	return d
}

func TestPostAttachmentRoundtrip(t *testing.T) {
	deps := newAttachmentDeps(t, 0) // 0 → default
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	// Upload
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", "hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(fw, bytes.NewReader([]byte("hi there"))); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/attachments", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("upload status %d: %s", rec.Code, rec.Body.String())
	}

	var desc struct {
		Hash string `json:"h"`
		Name string `json:"n"`
		Size int64  `json:"s"`
		Mime string `json:"m"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&desc); err != nil {
		t.Fatalf("decode descriptor: %v", err)
	}
	if desc.Size != 8 || desc.Name != "hello.txt" || desc.Hash == "" {
		t.Errorf("descriptor: %+v", desc)
	}

	// Download
	gReq := httptest.NewRequest(http.MethodGet, "/api/v1/attachments/"+desc.Hash+"/"+desc.Name, nil)
	gRec := httptest.NewRecorder()
	r.ServeHTTP(gRec, gReq)

	if gRec.Code != http.StatusOK {
		t.Fatalf("download status %d", gRec.Code)
	}
	if got := gRec.Body.String(); got != "hi there" {
		t.Errorf("download body = %q", got)
	}
	if cd := gRec.Header().Get("Content-Disposition"); cd == "" {
		t.Errorf("missing Content-Disposition header")
	}
	if got := gRec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q", got)
	}
}

func TestPostAttachmentTooLarge(t *testing.T) {
	deps := newAttachmentDeps(t, 8) // cap = 8 bytes
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", "big.bin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(make([]byte, 9)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/attachments", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
