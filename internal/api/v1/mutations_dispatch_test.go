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

// dispatch executes op against eng using the same MutateBoard + ApplyMutation
// pair that postMutation uses. Mirrors the production wiring so tests exercise
// the same atomic version-checked path.
func dispatch(eng *board.Engine, boardPath string, clientVersion int, op board.MutationOp) (*models.Board, error) {
	var out *models.Board
	err := eng.MutateBoard(boardPath, clientVersion, func(b *models.Board) error {
		if e := board.ApplyMutation(b, op); e != nil {
			return e
		}
		out = b
		return nil
	})
	return out, err
}

// newTwoColumnDeps returns a Deps with demo.md seeded with two columns ("Todo"
// and "Done"), each containing one card. Some ops (move_card, move_column,
// reorder_card) require at least two columns to make sensible calls.
func newTwoColumnDeps(t *testing.T) (v1.Deps, string) {
	t.Helper()
	deps := newTestDeps(t)
	path := filepath.Join(deps.Workspace.Dir, "demo.md")
	if err := deps.Engine.AddColumn(path, "Done"); err != nil {
		t.Fatalf("seed second column: %v", err)
	}
	if _, err := deps.Engine.AddCard(path, "Todo", "Second", false); err != nil {
		t.Fatalf("seed second card: %v", err)
	}
	return deps, path
}

func TestDispatchAllVariants(t *testing.T) {
	cases := []struct {
		name    string
		op      func(string) board.MutationOp
		deps    func(t *testing.T) (v1.Deps, string)
		wantErr bool
		errNote string
	}{
		// ── card ops ──────────────────────────────────────────────────────────
		{
			name: "add_card",
			op: func(_ string) board.MutationOp {
				return board.MutationOp{
					Type:    "add_card",
					AddCard: &board.AddCardOp{Column: "Todo", Title: "new card"},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
		{
			name: "add_card_prepend",
			op: func(_ string) board.MutationOp {
				return board.MutationOp{
					Type:    "add_card",
					AddCard: &board.AddCardOp{Column: "Todo", Title: "prepended", Prepend: true},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
		{
			name: "move_card",
			op: func(_ string) board.MutationOp {
				return board.MutationOp{
					Type:     "move_card",
					MoveCard: &board.MoveCardOp{ColIdx: 0, CardIdx: 0, TargetColumn: "Done"},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				d, p := newTwoColumnDeps(t)
				return d, p
			},
		},
		{
			name: "reorder_card",
			op: func(_ string) board.MutationOp {
				return board.MutationOp{
					Type: "reorder_card",
					ReorderCard: &board.ReorderCardOp{
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
			op: func(_ string) board.MutationOp {
				return board.MutationOp{
					Type: "edit_card",
					EditCard: &board.EditCardOp{
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
			op: func(_ string) board.MutationOp {
				return board.MutationOp{
					Type:       "delete_card",
					DeleteCard: &board.DeleteCardOp{ColIdx: 0, CardIdx: 0},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
		{
			name: "complete_card",
			op: func(_ string) board.MutationOp {
				return board.MutationOp{
					Type:         "complete_card",
					CompleteCard: &board.CompleteCardOp{ColIdx: 0, CardIdx: 0},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
		{
			name: "tag_card",
			op: func(_ string) board.MutationOp {
				return board.MutationOp{
					Type:    "tag_card",
					TagCard: &board.TagCardOp{ColIdx: 0, CardIdx: 0, Tags: []string{"urgent"}},
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
			op: func(_ string) board.MutationOp {
				return board.MutationOp{
					Type:      "add_column",
					AddColumn: &board.AddColumnOp{Name: "Backlog"},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
		{
			name: "rename_column",
			op: func(_ string) board.MutationOp {
				return board.MutationOp{
					Type:         "rename_column",
					RenameColumn: &board.RenameColumnOp{OldName: "Todo", NewName: "Doing"},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
		{
			name: "delete_column",
			op: func(_ string) board.MutationOp {
				return board.MutationOp{
					Type:         "delete_column",
					DeleteColumn: &board.DeleteColumnOp{Name: "Done"},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				d, p := newTwoColumnDeps(t)
				return d, p
			},
		},
		{
			name: "move_column",
			op: func(_ string) board.MutationOp {
				return board.MutationOp{
					Type:       "move_column",
					MoveColumn: &board.MoveColumnOp{Name: "Done", AfterCol: ""},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				d, p := newTwoColumnDeps(t)
				return d, p
			},
		},
		{
			name: "sort_column",
			op: func(_ string) board.MutationOp {
				return board.MutationOp{
					Type:       "sort_column",
					SortColumn: &board.SortColumnOp{ColIdx: 0, SortBy: "priority"},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
		{
			name: "toggle_column_collapse",
			op: func(_ string) board.MutationOp {
				return board.MutationOp{
					Type:                 "toggle_column_collapse",
					ToggleColumnCollapse: &board.ToggleColumnCollapseOp{ColIdx: 0},
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
			op: func(_ string) board.MutationOp {
				return board.MutationOp{
					Type: "update_board_meta",
					UpdateBoardMeta: &board.UpdateBoardMetaOp{
						Name:        "Renamed",
						Description: "desc",
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
			op: func(_ string) board.MutationOp {
				return board.MutationOp{
					Type:               "update_board_members",
					UpdateBoardMembers: &board.UpdateBoardMembersOp{Members: []string{"alice", "bob"}},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
		{
			name: "update_board_icon",
			op: func(_ string) board.MutationOp {
				icon := "🚀"
				return board.MutationOp{
					Type:            "update_board_icon",
					UpdateBoardIcon: &board.UpdateBoardIconOp{Icon: &icon},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
		{
			name: "update_board_settings",
			op: func(_ string) board.MutationOp {
				show := true
				return board.MutationOp{
					Type: "update_board_settings",
					UpdateBoardSettings: &board.UpdateBoardSettingsOp{
						Settings: models.BoardSettings{ShowCheckbox: &show},
					},
				}
			},
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},

		// ── attachment ops ────────────────────────────────────────────────────
		{
			name: "add_attachments",
			op: func(_ string) board.MutationOp {
				return board.MutationOp{
					Type: "add_attachments",
					AddAttachments: &board.AddAttachmentsOp{
						ColIdx: 0, CardIdx: 0,
						Items: []models.Attachment{{Hash: "h.pdf", Name: "n.pdf", Size: 1, Mime: "application/pdf"}},
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
			op:      func(_ string) board.MutationOp { return board.MutationOp{Type: "bogus"} },
			wantErr: true,
			errNote: "dispatch must reject unknown op types",
			deps: func(t *testing.T) (v1.Deps, string) {
				deps := newTestDeps(t)
				return deps, filepath.Join(deps.Workspace.Dir, "demo.md")
			},
		},
		{
			name:    "add_card_nil_params_errors",
			op:      func(_ string) board.MutationOp { return board.MutationOp{Type: "add_card"} },
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
			_, err := dispatch(deps.Engine, path, -1, op)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("wanted error (%s) but got nil", tc.errNote)
				}
				return
			}
			if err != nil {
				t.Fatalf("dispatch %q: %v", tc.name, err)
			}

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
				if !strings.Contains(string(raw), "icon:") {
					t.Errorf("update_board_icon: 'icon:' key not found in file")
				}
			case "add_attachments":
				raw, _ := os.ReadFile(path)
				if !strings.Contains(string(raw), "attachments:") || !strings.Contains(string(raw), op.AddAttachments.Items[0].Hash) {
					t.Errorf("add_attachments: attachments line / hash not found in file:\n%s", raw)
				}
			}
		})
	}
}

func TestApply_moveCardToBoard(t *testing.T) {
	deps := newTestDeps(t)
	path := filepath.Join(deps.Workspace.Dir, "demo.md")
	op := board.MutationOp{
		Type: "move_card_to_board",
		MoveCardToBoard: &board.MoveCardToBoardOp{
			ColIdx: 0, CardIdx: 0, DstBoard: "other", DstColumn: "Inbox",
		},
	}
	b, err := dispatch(deps.Engine, path, -1, op)
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if b == nil {
		t.Fatal("expected non-nil board")
	}
	if len(b.Columns) == 0 || len(b.Columns[0].Cards) != 0 {
		t.Errorf("expected card removed from source, got columns=%+v", b.Columns)
	}
}

func TestApply_nilParams(t *testing.T) {
	b := &models.Board{Columns: []models.Column{{Name: "Todo", Cards: []models.Card{{Title: "x"}}}}}
	for _, typ := range board.MutationVariantNames() {
		t.Run(typ, func(t *testing.T) {
			op := board.MutationOp{Type: typ}
			if err := board.ApplyMutation(b, op); err == nil {
				t.Errorf("want error for nil params on %q, got nil", typ)
			}
		})
	}
}

func TestDispatchRespectsClientVersion(t *testing.T) {
	deps := newTestDeps(t)
	path := filepath.Join(deps.Workspace.Dir, "demo.md")

	op := board.MutationOp{
		Type:    "add_card",
		AddCard: &board.AddCardOp{Column: "Todo", Title: "v-fail"},
	}
	_, err := dispatch(deps.Engine, path, 0, op)
	if err == nil {
		t.Fatal("want ErrVersionConflict, got nil")
	}
	if !errors.Is(err, board.ErrVersionConflict) {
		t.Errorf("want ErrVersionConflict, got %v", err)
	}
}
