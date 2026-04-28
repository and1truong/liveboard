package parity_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/pkg/models"
)

type vector struct {
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	BoardBefore   json.RawMessage `json:"board_before"`
	Op            json.RawMessage `json:"op"`
	BoardAfter    json.RawMessage `json:"board_after,omitempty"`
	ExpectedError string          `json:"expected_error,omitempty"`
}

func TestVectorSuite(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "mutations")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read testdata dir: %v", err)
	}

	found := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		found++
		path := filepath.Join(root, e.Name())
		t.Run(e.Name(), func(t *testing.T) {
			runVector(t, path)
		})
	}
	if found == 0 {
		t.Fatal("no vectors found")
	}
}

func runVector(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var vec vector
	if err = json.Unmarshal(raw, &vec); err != nil {
		t.Fatalf("parse vector: %v", err)
	}

	var b models.Board
	if err = json.Unmarshal(vec.BoardBefore, &b); err != nil {
		t.Fatalf("parse board_before: %v", err)
	}

	var op board.MutationOp
	if err = json.Unmarshal(vec.Op, &op); err != nil {
		t.Fatalf("parse op: %v", err)
	}

	applyErr := board.ApplyMutation(&b, op)

	if vec.ExpectedError != "" {
		if applyErr == nil {
			t.Fatalf("want error %q, got nil", vec.ExpectedError)
		}
		if got := sentinelCode(applyErr); got != vec.ExpectedError {
			t.Fatalf("want error %q, got %q (%v)", vec.ExpectedError, got, applyErr)
		}
		return
	}

	if applyErr != nil {
		t.Fatalf("unexpected error: %v", applyErr)
	}

	// Normalize by round-tripping both sides through models.Board so that
	// zero-valued struct fields appear on both sides uniformly.
	var wantBoard models.Board
	if err = json.Unmarshal(vec.BoardAfter, &wantBoard); err != nil {
		t.Fatalf("parse board_after: %v", err)
	}
	wantJSON, err := json.Marshal(&wantBoard)
	if err != nil {
		t.Fatalf("marshal want: %v", err)
	}
	gotJSON, err := json.Marshal(&b)
	if err != nil {
		t.Fatalf("marshal got: %v", err)
	}

	var want, got any
	if err := json.Unmarshal(wantJSON, &want); err != nil {
		t.Fatalf("re-parse want: %v", err)
	}
	if err := json.Unmarshal(gotJSON, &got); err != nil {
		t.Fatalf("re-parse got: %v", err)
	}
	want = stripNulls(stripCardIDs(want))
	got = stripNulls(stripCardIDs(got))

	if diff := jsonDiff(want, got); diff != "" {
		t.Errorf("board mismatch:\n%s", diff)
	}
}

func sentinelCode(err error) string {
	switch {
	case errors.Is(err, board.ErrNotFound):
		return "NOT_FOUND"
	case errors.Is(err, board.ErrOutOfRange):
		return "OUT_OF_RANGE"
	case errors.Is(err, board.ErrInvalidInput):
		return "INVALID"
	default:
		return "INTERNAL"
	}
}

// stripCardIDs recursively removes "id" keys from card objects so that
// mutation vector tests are not sensitive to ID assignment (covered separately
// in board_test.go).
func stripCardIDs(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, val := range x {
			if k == "id" {
				continue
			}
			out[k] = stripCardIDs(val)
		}
		return out
	case []any:
		out := make([]any, 0, len(x))
		for _, el := range x {
			out = append(out, stripCardIDs(el))
		}
		return out
	default:
		return v
	}
}

// stripNulls recursively removes keys whose value is nil and drops nil elements
// from slices, so "null" / missing / "[]" compare equal across runners.
func stripNulls(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, val := range x {
			if val == nil {
				continue
			}
			stripped := stripNulls(val)
			// Treat empty slices the same as nil/missing.
			if arr, ok := stripped.([]any); ok && len(arr) == 0 {
				continue
			}
			out[k] = stripped
		}
		return out
	case []any:
		out := make([]any, 0, len(x))
		for _, el := range x {
			if el == nil {
				continue
			}
			out = append(out, stripNulls(el))
		}
		return out
	default:
		return v
	}
}

func jsonDiff(want, got any) string {
	wb, _ := json.MarshalIndent(want, "", "  ")
	gb, _ := json.MarshalIndent(got, "", "  ")
	if string(wb) == string(gb) {
		return ""
	}
	return "want:\n" + string(wb) + "\n\ngot:\n" + string(gb)
}
