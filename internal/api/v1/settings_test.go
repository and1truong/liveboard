package v1_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	v1 "github.com/and1truong/liveboard/internal/api/v1"
)

func TestGetBoardSettings(t *testing.T) {
	deps := newTestDeps(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/settings/demo", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resolved struct {
		ViewMode string `json:"view_mode"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resolved); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resolved.ViewMode == "" {
		t.Errorf("want non-empty view_mode in resolved settings, got empty string")
	}
}

func TestPutBoardSettings(t *testing.T) {
	deps := newTestDeps(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	patch := `{"view_mode":"calendar"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/boards/settings/demo",
		bytes.NewReader([]byte(patch)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify persistence: GET the board and check settings.
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/boards/board/demo", nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	var b struct {
		Settings struct {
			ViewMode *string `json:"view_mode"`
		} `json:"settings"`
	}
	_ = json.NewDecoder(rec2.Body).Decode(&b)
	if b.Settings.ViewMode == nil || *b.Settings.ViewMode != "calendar" {
		t.Errorf("view_mode not persisted: %+v", b.Settings.ViewMode)
	}
}

func TestGetBoardSettings_notFound(t *testing.T) {
	deps := newTestDeps(t)
	rec, body := doReq(t, deps, http.MethodGet, "/api/v1/boards/settings/nope", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d: %s", rec.Code, body)
	}
}

func TestPutBoardSettings_notFound(t *testing.T) {
	deps := newTestDeps(t)
	rec, body := doReq(t, deps, http.MethodPut, "/api/v1/boards/settings/nope", `{"view_mode":"board"}`)
	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d: %s", rec.Code, body)
	}
}

func TestPutBoardSettings_badJSON(t *testing.T) {
	deps := newTestDeps(t)
	rec, body := doReq(t, deps, http.MethodPut, "/api/v1/boards/settings/demo", "not json")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d: %s", rec.Code, body)
	}
}

func TestPutBoardSettings_withSSE(t *testing.T) {
	deps := newTestDepsWithSSE(t)
	rec, body := doReq(t, deps, http.MethodPut, "/api/v1/boards/settings/demo", `{"view_mode":"list"}`)
	if rec.Code != http.StatusNoContent {
		t.Errorf("want 204, got %d: %s", rec.Code, body)
	}
}

// TestPutBoardSettings_preservesUntouchedOverride pins the patch-merge
// semantic: a follow-up PUT that omits a field MUST NOT clear an override
// set by an earlier PUT. The pre-fix handler did a true-replace and silently
// nuked any override the renderer didn't include in its payload — the
// BoardSettingsModal omits card_position from its 5-field write, so any
// per-board card_position override was wiped on every save.
func TestPutBoardSettings_preservesUntouchedOverride(t *testing.T) {
	deps := newTestDeps(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	// First PUT sets card_position = "prepend".
	put := func(body string) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPut, "/api/v1/boards/settings/demo",
			bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("PUT %s: want 204, got %d: %s", body, rec.Code, rec.Body.String())
		}
	}

	put(`{"card_position":"prepend"}`)
	// Second PUT mirrors what BoardSettingsModal sends — 5 fields, no card_position.
	put(`{"show_checkbox":true,"expand_columns":false,"card_display_mode":"compact","view_mode":"list","week_start":"monday"}`)

	// Read the raw board to inspect overrides.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/board/demo", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var b struct {
		Settings struct {
			CardPosition *string `json:"card_position"`
			ViewMode     *string `json:"view_mode"`
		} `json:"settings"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&b); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if b.Settings.CardPosition == nil || *b.Settings.CardPosition != "prepend" {
		t.Fatalf("card_position override was cleared by second PUT: %+v", b.Settings.CardPosition)
	}
	if b.Settings.ViewMode == nil || *b.Settings.ViewMode != "list" {
		t.Fatalf("view_mode from second PUT not applied: %+v", b.Settings.ViewMode)
	}
}

// TestPutBoardSettings_dropsInvalidEnum confirms SanitizeBoardSettings runs
// before the patch-merge: an invalid enum value is dropped (not stored as
// a malformed override).
func TestPutBoardSettings_dropsInvalidEnum(t *testing.T) {
	deps := newTestDeps(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	body := `{"view_mode":"garbage","week_start":"funday","card_position":"sideways"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/boards/settings/demo",
		bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d: %s", rec.Code, rec.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/boards/board/demo", nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	var b struct {
		Settings struct {
			ViewMode     *string `json:"view_mode"`
			WeekStart    *string `json:"week_start"`
			CardPosition *string `json:"card_position"`
		} `json:"settings"`
	}
	_ = json.NewDecoder(rec2.Body).Decode(&b)
	if b.Settings.ViewMode != nil {
		t.Errorf("invalid view_mode persisted: %q", *b.Settings.ViewMode)
	}
	if b.Settings.WeekStart != nil {
		t.Errorf("invalid week_start persisted: %q", *b.Settings.WeekStart)
	}
	if b.Settings.CardPosition != nil {
		t.Errorf("invalid card_position persisted: %q", *b.Settings.CardPosition)
	}
}
