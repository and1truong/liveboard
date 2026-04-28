package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/and1truong/liveboard/internal/workspace"
)

func TestBasicAuth(t *testing.T) {
	dir := t.TempDir()
	ws := workspace.Open(dir)

	t.Run("enabled rejects unauthenticated", func(t *testing.T) {
		srv := NewServer(ws, ws.Engine, false, false, false, "test", "admin", "secret")
		ts := httptest.NewServer(srv.Router())
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/")
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("enabled accepts valid credentials", func(t *testing.T) {
		srv := NewServer(ws, ws.Engine, false, false, false, "test", "admin", "secret")
		ts := httptest.NewServer(srv.Router())
		defer ts.Close()

		req, _ := http.NewRequest("GET", ts.URL+"/", nil)
		req.SetBasicAuth("admin", "secret")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusUnauthorized {
			t.Fatal("expected authenticated request to succeed, got 401")
		}
	})

	t.Run("enabled rejects wrong password", func(t *testing.T) {
		srv := NewServer(ws, ws.Engine, false, false, false, "test", "admin", "secret")
		ts := httptest.NewServer(srv.Router())
		defer ts.Close()

		req, _ := http.NewRequest("GET", ts.URL+"/", nil)
		req.SetBasicAuth("admin", "wrong")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("disabled allows unauthenticated", func(t *testing.T) {
		srv := NewServer(ws, ws.Engine, false, false, false, "test", "", "")
		ts := httptest.NewServer(srv.Router())
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/")
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusUnauthorized {
			t.Fatal("expected no auth when credentials empty, got 401")
		}
	})
}
