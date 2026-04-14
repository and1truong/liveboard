package v1_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	v1 "github.com/and1truong/liveboard/internal/api/v1"
	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/pkg/models"
)

// newTwoColumnDeps returns a Deps with demo.md seeded with two columns ("Todo"
// and "Done"), each containing one card. Some ops (move_card, move_column,
// reorder_card) require at least two columns to make sensible calls.
func newTwoColumnDeps(t *testing.T) (v1.Deps, string) {
	t.Helper()
	deps := newTestDeps(t)
	path := filepath.Join(deps.Workspace.Dir, "demo.md")
	// Add a second column so move/reorder ops have a valid target.
	if err := deps.Engine.AddColumn(path, "Done"); err != nil {
		t.Fatalf("seed second column: %v", err)
	}
	// Add a second card to Todo so reorder has a non-trivial before_idx.
	if _, err := deps.Engine.AddCard(path, "Todo", "Second", false); err != nil {
		t.Fatalf("seed second card: %v", err)
	}
	return deps, path
}

func TestDispatchAllVariants(t *testing.T) {
	cases := []struct {
		name    string
		op      func(string) v1.MutationOp
		deps    func(t *testing.T) (v1.Deps, string)
		wantErr bool
		errNote string // why we expect an error
	}{
		// ── card ops ──────────────────────────────────────────────────────────
		{
			name: "add_card",
			op: func(_ string) v1.MutationOp {
				return v1.MutationOp{
					Type:    "add_card",
					AddCard: &v1.AddCardOp{Column: "Todo", Title: "new card"},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
		{
			name: "add_card_prepend",
			op: func(_ string) v1.MutationOp {
				return v1.MutationOp{
					Type:    "add_card",
					AddCard: &v1.AddCardOp{Column: "Todo", Title: "prepended", Prepend: true},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
		{
			name: "move_card",
			op: func(_ string) v1.MutationOp {
				// Move card 0 from column 0 (Todo) to column 1 (Done).
				return v1.MutationOp{
					Type:     "move_card",
					MoveCard: &v1.MoveCardOp{ColIdx: 0, CardIdx: 0, TargetColumn: "Done"},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				d, p := newTwoColumnDeps(t)
				return d, p
			},
		},
		{
			name: "reorder_card",
			op: func(_ string) v1.MutationOp {
				// Two cards in Todo (indices 0,1); move card 1 before index 0.
				return v1.MutationOp{
					Type: "reorder_card",
					ReorderCard: &v1.ReorderCardOp{
						ColIdx:       0,
						CardIdx:      1,
						BeforeIdx:    0,
						TargetColumn: "Todo",
					},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				d, p := newTwoColumnDeps(t)
				return d, p
			},
		},
		{
			name: "edit_card",
			op: func(_ string) v1.MutationOp {
				return v1.MutationOp{
					Type: "edit_card",
					EditCard: &v1.EditCardOp{
						ColIdx:   0,
						CardIdx:  0,
						Title:    "edited title",
						Body:     "some body",
						Tags:     []string{"go"},
						Priority: "high",
						Due:      "2026-12-31",
						Assignee: "alice",
					},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
		{
			name: "delete_card",
			op: func(_ string) v1.MutationOp {
				return v1.MutationOp{
					Type:       "delete_card",
					DeleteCard: &v1.DeleteCardOp{ColIdx: 0, CardIdx: 0},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
		{
			name: "complete_card",
			op: func(_ string) v1.MutationOp {
				return v1.MutationOp{
					Type:         "complete_card",
					CompleteCard: &v1.CompleteCardOp{ColIdx: 0, CardIdx: 0},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
		{
			name: "tag_card",
			op: func(_ string) v1.MutationOp {
				return v1.MutationOp{
					Type:    "tag_card",
					TagCard: &v1.TagCardOp{ColIdx: 0, CardIdx: 0, Tags: []string{"urgent"}},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},

		// ── column ops ────────────────────────────────────────────────────────
		{
			name: "add_column",
			op: func(_ string) v1.MutationOp {
				return v1.MutationOp{
					Type:      "add_column",
					AddColumn: &v1.AddColumnOp{Name: "Backlog"},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
		{
			name: "rename_column",
			op: func(_ string) v1.MutationOp {
				return v1.MutationOp{
					Type:         "rename_column",
					RenameColumn: &v1.RenameColumnOp{OldName: "Todo", NewName: "Doing"},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
		{
			// delete_column: seed has only "Todo"; after adding "Done" we delete
			// "Done" so the board still has at least one column afterwards.
			name: "delete_column",
			op: func(_ string) v1.MutationOp {
				return v1.MutationOp{
					Type:         "delete_column",
					DeleteColumn: &v1.DeleteColumnOp{Name: "Done"},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				d, p := newTwoColumnDeps(t)
				return d, p
			},
		},
		{
			// move_column: board has Todo(0) and Done(1); move Done after "" (i.e.
			// to front) to make it Done(0), Todo(1).
			name: "move_column",
			op: func(_ string) v1.MutationOp {
				return v1.MutationOp{
					Type:       "move_column",
					MoveColumn: &v1.MoveColumnOp{Name: "Done", AfterCol: ""},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				d, p := newTwoColumnDeps(t)
				return d, p
			},
		},
		{
			name: "sort_column",
			op: func(_ string) v1.MutationOp {
				return v1.MutationOp{
					Type:       "sort_column",
					SortColumn: &v1.SortColumnOp{ColIdx: 0, SortBy: "priority"},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
		{
			name: "toggle_column_collapse",
			op: func(_ string) v1.MutationOp {
				return v1.MutationOp{
					Type:                 "toggle_column_collapse",
					ToggleColumnCollapse: &v1.ToggleColumnCollapseOp{ColIdx: 0},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},

		// ── board meta ops ────────────────────────────────────────────────────
		{
			name: "update_board_meta",
			op: func(_ string) v1.MutationOp {
				return v1.MutationOp{
					Type: "update_board_meta",
					UpdateBoardMeta: &v1.UpdateBoardMetaOp{
						Name:        "Renamed",
						Description: "desc",
						Tags:        []string{"q1"},
					},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
		{
			name: "update_board_members",
			op: func(_ string) v1.MutationOp {
				return v1.MutationOp{
					Type:               "update_board_members",
					UpdateBoardMembers: &v1.UpdateBoardMembersOp{Members: []string{"alice", "bob"}},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
		{
			name: "update_board_icon",
			op: func(_ string) v1.MutationOp {
				return v1.MutationOp{
					Type:            "update_board_icon",
					UpdateBoardIcon: &v1.UpdateBoardIconOp{Icon: "🚀"},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
		{
			name: "update_board_settings",
			op: func(_ string) v1.MutationOp {
				show := true
				return v1.MutationOp{
					Type: "update_board_settings",
					UpdateBoardSettings: &v1.UpdateBoardSettingsOp{
						Settings: models.BoardSettings{ShowCheckbox: &show},
					},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},

		// ── error cases ───────────────────────────────────────────────────────
		{
			name:    "unknown_op_errors",
			op:      func(_ string) v1.MutationOp { return v1.MutationOp{Type: "bogus"} },
			wantErr: true,
			errNote: "Dispatch must reject unknown op types",
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
		{
			name:    "add_card_nil_params_errors",
			op:      func(_ string) v1.MutationOp { return v1.MutationOp{Type: "add_card"} },
			wantErr: true,
			errNote: "nil params should be caught before engine call",
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deps, path := tc.deps(t)
			op := tc.op(path)
			_, err := v1.Dispatch(deps.Engine, path, -1, op)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("wanted error (%s) but got nil", tc.errNote)
				}
				return
			}
			if err != nil {
				t.Fatalf("dispatch %q: %v", tc.name, err)
			}

			// Spot-check that the file was actually written for a few key ops.
			switch op.Type {
			case "add_card":
				raw, _ := os.ReadFile(path)
				if !strings.Contains(string(raw), op.AddCard.Title) {
					t.Errorf("add_card: title %q not found in file", op.AddCard.Title)
				}
			case "rename_column":
				raw, _ := os.ReadFile(path)
				if !strings.Contains(string(raw), op.RenameColumn.NewName) {
					t.Errorf("rename_column: new name %q not found in file", op.RenameColumn.NewName)
				}
			case "update_board_meta":
				raw, _ := os.ReadFile(path)
				if !strings.Contains(string(raw), op.UpdateBoardMeta.Name) {
					t.Errorf("update_board_meta: name %q not found in file", op.UpdateBoardMeta.Name)
				}
			case "update_board_icon":
				raw, _ := os.ReadFile(path)
				// The YAML library escapes multi-byte emoji (e.g. 🚀 → "\U0001F680"),
				// so check for the yaml key rather than the literal rune.
				if !strings.Contains(string(raw), "icon:") {
					t.Errorf("update_board_icon: 'icon:' key not found in file")
				}
			}
		})
	}
}

func TestDispatchRespectsClientVersion(t *testing.T) {
	deps := newTestDeps(t)
	path := filepath.Join(deps.Workspace.Dir, "demo.md")

	// Stale version should fail with ErrVersionConflict.
	// Seed board is at version 1; passing version 0 must be rejected.
	op := v1.MutationOp{
		Type:    "add_card",
		AddCard: &v1.AddCardOp{Column: "Todo", Title: "v-fail"},
	}
	_, err := v1.Dispatch(deps.Engine, path, 0, op)
	if err == nil {
		t.Fatal("want ErrVersionConflict, got nil")
	}
	if !errors.Is(err, board.ErrVersionConflict) {
		t.Errorf("want ErrVersionConflict, got %v", err)
	}
}
