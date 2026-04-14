package v1_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	v1 "github.com/and1truong/liveboard/internal/api/v1"
)

func TestVersionsHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/versions", nil)
	rec := httptest.NewRecorder()
	v1.VersionsHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}

	var body struct {
		Supported []string `json:"supported"`
		Current   string   `json:"current"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Current != "v1" {
		t.Errorf("want current=v1, got %q", body.Current)
	}
	if len(body.Supported) != 1 || body.Supported[0] != "v1" {
		t.Errorf("want supported=[v1], got %v", body.Supported)
	}
}
