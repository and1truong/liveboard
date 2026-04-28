package board_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/pkg/models"
)

func TestMutationOpUnmarshalAddCard(t *testing.T) {
	raw := []byte(`{"type":"add_card","column":"Todo","title":"hello","prepend":false}`)
	var op board.MutationOp
	if err := json.Unmarshal(raw, &op); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if op.Type != "add_card" {
		t.Errorf("want type=add_card, got %q", op.Type)
	}
	if op.AddCard == nil {
		t.Fatal("AddCard params should be populated")
	}
	if op.AddCard.Title != "hello" {
		t.Errorf("want title=hello, got %q", op.AddCard.Title)
	}
}

func TestMutationOpUnmarshalMoveCard(t *testing.T) {
	raw := []byte(`{"type":"move_card","col_idx":0,"card_idx":1,"target_column":"Done"}`)
	var op board.MutationOp
	if err := json.Unmarshal(raw, &op); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if op.MoveCard == nil {
		t.Fatal("MoveCard params should be populated")
	}
	if op.MoveCard.TargetColumn != "Done" {
		t.Errorf("want target=Done, got %q", op.MoveCard.TargetColumn)
	}
}

func TestMutationOpUnmarshalUnknownType(t *testing.T) {
	raw := []byte(`{"type":"not_a_real_op"}`)
	var op board.MutationOp
	if err := json.Unmarshal(raw, &op); err == nil {
		t.Fatal("want error for unknown op type")
	}
}

func TestMutationOpUnmarshalAllVariants(t *testing.T) {
	cases := []struct {
		name        string
		jsonStr     string
		wantType    string
		nonNilField func(*board.MutationOp) bool
	}{
		{
			name:        "add_card",
			jsonStr:     `{"type":"add_card","column":"Todo","title":"x"}`,
			wantType:    "add_card",
			nonNilField: func(m *board.MutationOp) bool { return m.AddCard != nil },
		},
		{
			name:        "move_card",
			jsonStr:     `{"type":"move_card","col_idx":0,"card_idx":1,"target_column":"Done"}`,
			wantType:    "move_card",
			nonNilField: func(m *board.MutationOp) bool { return m.MoveCard != nil },
		},
		{
			name:        "reorder_card",
			jsonStr:     `{"type":"reorder_card","col_idx":0,"card_idx":1,"before_idx":0,"target_column":"Todo"}`,
			wantType:    "reorder_card",
			nonNilField: func(m *board.MutationOp) bool { return m.ReorderCard != nil },
		},
		{
			name:        "edit_card",
			jsonStr:     `{"type":"edit_card","col_idx":0,"card_idx":0,"title":"t","body":"b","tags":[],"priority":"high","due":"","assignee":""}`,
			wantType:    "edit_card",
			nonNilField: func(m *board.MutationOp) bool { return m.EditCard != nil },
		},
		{
			name:        "delete_card",
			jsonStr:     `{"type":"delete_card","col_idx":0,"card_idx":0}`,
			wantType:    "delete_card",
			nonNilField: func(m *board.MutationOp) bool { return m.DeleteCard != nil },
		},
		{
			name:        "complete_card",
			jsonStr:     `{"type":"complete_card","col_idx":0,"card_idx":0}`,
			wantType:    "complete_card",
			nonNilField: func(m *board.MutationOp) bool { return m.CompleteCard != nil },
		},
		{
			name:        "tag_card",
			jsonStr:     `{"type":"tag_card","col_idx":0,"card_idx":0,"tags":["x"]}`,
			wantType:    "tag_card",
			nonNilField: func(m *board.MutationOp) bool { return m.TagCard != nil },
		},
		{
			name:        "add_column",
			jsonStr:     `{"type":"add_column","name":"Backlog"}`,
			wantType:    "add_column",
			nonNilField: func(m *board.MutationOp) bool { return m.AddColumn != nil },
		},
		{
			name:        "rename_column",
			jsonStr:     `{"type":"rename_column","old_name":"Todo","new_name":"Doing"}`,
			wantType:    "rename_column",
			nonNilField: func(m *board.MutationOp) bool { return m.RenameColumn != nil },
		},
		{
			name:        "delete_column",
			jsonStr:     `{"type":"delete_column","name":"Done"}`,
			wantType:    "delete_column",
			nonNilField: func(m *board.MutationOp) bool { return m.DeleteColumn != nil },
		},
		{
			name:        "move_column",
			jsonStr:     `{"type":"move_column","name":"Done","after_col":"Todo"}`,
			wantType:    "move_column",
			nonNilField: func(m *board.MutationOp) bool { return m.MoveColumn != nil },
		},
		{
			name:        "sort_column",
			jsonStr:     `{"type":"sort_column","col_idx":0,"sort_by":"priority"}`,
			wantType:    "sort_column",
			nonNilField: func(m *board.MutationOp) bool { return m.SortColumn != nil },
		},
		{
			name:        "toggle_column_collapse",
			jsonStr:     `{"type":"toggle_column_collapse","col_idx":2}`,
			wantType:    "toggle_column_collapse",
			nonNilField: func(m *board.MutationOp) bool { return m.ToggleColumnCollapse != nil },
		},
		{
			name:        "update_board_meta",
			jsonStr:     `{"type":"update_board_meta","name":"X","description":""}`,
			wantType:    "update_board_meta",
			nonNilField: func(m *board.MutationOp) bool { return m.UpdateBoardMeta != nil },
		},
		{
			name:        "update_board_members",
			jsonStr:     `{"type":"update_board_members","members":["alice"]}`,
			wantType:    "update_board_members",
			nonNilField: func(m *board.MutationOp) bool { return m.UpdateBoardMembers != nil },
		},
		{
			name:        "update_board_icon",
			jsonStr:     `{"type":"update_board_icon","icon":"🚀"}`,
			wantType:    "update_board_icon",
			nonNilField: func(m *board.MutationOp) bool { return m.UpdateBoardIcon != nil },
		},
		{
			name:        "update_board_settings",
			jsonStr:     `{"type":"update_board_settings","settings":{}}`,
			wantType:    "update_board_settings",
			nonNilField: func(m *board.MutationOp) bool { return m.UpdateBoardSettings != nil },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var op board.MutationOp
			if err := json.Unmarshal([]byte(tc.jsonStr), &op); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if op.Type != tc.wantType {
				t.Errorf("want type=%q, got %q", tc.wantType, op.Type)
			}
			if !tc.nonNilField(&op) {
				t.Errorf("expected non-nil variant field for type=%q", tc.wantType)
			}
		})
	}
}

func TestMutationOpRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		op   board.MutationOp
	}{
		{
			name: "add_card",
			op: board.MutationOp{
				Type:    "add_card",
				AddCard: &board.AddCardOp{Column: "Todo", Title: "my card", Prepend: true},
			},
		},
		{
			name: "move_card",
			op: board.MutationOp{
				Type:     "move_card",
				MoveCard: &board.MoveCardOp{ColIdx: 1, CardIdx: 2, TargetColumn: "Done"},
			},
		},
		{
			name: "edit_card",
			op: board.MutationOp{
				Type: "edit_card",
				EditCard: &board.EditCardOp{
					ColIdx: 0, CardIdx: 3,
					Title: "edited", Body: "body text",
					Tags: []string{"go", "api"}, Priority: "high",
					Due: "2026-12-31", Assignee: "alice",
				},
			},
		},
		{
			name: "update_board_settings",
			op: board.MutationOp{
				Type: "update_board_settings",
				UpdateBoardSettings: &board.UpdateBoardSettingsOp{
					Settings: models.BoardSettings{ShowCheckbox: mutBoolPtr(true)},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.op)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var got board.MutationOp
			if jerr := json.Unmarshal(data, &got); jerr != nil {
				t.Fatalf("unmarshal: %v", jerr)
			}
			if got.Type != tc.op.Type {
				t.Errorf("type mismatch: want %q got %q", tc.op.Type, got.Type)
			}
			data2, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("re-marshal: %v", err)
			}
			var m1, m2 map[string]json.RawMessage
			_ = json.Unmarshal(data, &m1)
			_ = json.Unmarshal(data2, &m2)
			for k, v := range m1 {
				v2, ok := m2[k]
				if !ok {
					t.Errorf("key %q missing in second marshal", k)
					continue
				}
				if string(v) != string(v2) {
					t.Errorf("key %q: want %s got %s", k, v, v2)
				}
			}
		})
	}
}

func mutBoolPtr(b bool) *bool    { return &b }
func mutStrPtr(s string) *string { return &s }

func TestMutationOpMarshalJSON_allVariants(t *testing.T) {
	cases := []board.MutationOp{
		{Type: "add_card", AddCard: &board.AddCardOp{Column: "Todo", Title: "x", Prepend: true}},
		{Type: "move_card", MoveCard: &board.MoveCardOp{ColIdx: 0, CardIdx: 1, TargetColumn: "Done"}},
		{Type: "reorder_card", ReorderCard: &board.ReorderCardOp{ColIdx: 0, CardIdx: 1, BeforeIdx: 0, TargetColumn: "Todo"}},
		{Type: "edit_card", EditCard: &board.EditCardOp{ColIdx: 0, CardIdx: 0, Title: "t", Priority: "high"}},
		{Type: "delete_card", DeleteCard: &board.DeleteCardOp{ColIdx: 0, CardIdx: 0}},
		{Type: "complete_card", CompleteCard: &board.CompleteCardOp{ColIdx: 0, CardIdx: 0}},
		{Type: "tag_card", TagCard: &board.TagCardOp{ColIdx: 0, CardIdx: 0, Tags: []string{"x"}}},
		{Type: "add_column", AddColumn: &board.AddColumnOp{Name: "Backlog"}},
		{Type: "rename_column", RenameColumn: &board.RenameColumnOp{OldName: "Todo", NewName: "Doing"}},
		{Type: "delete_column", DeleteColumn: &board.DeleteColumnOp{Name: "Done"}},
		{Type: "move_column", MoveColumn: &board.MoveColumnOp{Name: "Done", AfterCol: "Todo"}},
		{Type: "sort_column", SortColumn: &board.SortColumnOp{ColIdx: 0, SortBy: "priority"}},
		{Type: "toggle_column_collapse", ToggleColumnCollapse: &board.ToggleColumnCollapseOp{ColIdx: 1}},
		{Type: "update_board_meta", UpdateBoardMeta: &board.UpdateBoardMetaOp{Name: "X"}},
		{Type: "update_board_members", UpdateBoardMembers: &board.UpdateBoardMembersOp{Members: []string{"alice"}}},
		{Type: "update_board_icon", UpdateBoardIcon: &board.UpdateBoardIconOp{Icon: mutStrPtr("🎯")}},
		{Type: "update_board_settings", UpdateBoardSettings: &board.UpdateBoardSettingsOp{Settings: models.BoardSettings{ShowCheckbox: mutBoolPtr(true)}}},
		{Type: "move_card_to_board", MoveCardToBoard: &board.MoveCardToBoardOp{ColIdx: 0, CardIdx: 0, DstBoard: "other", DstColumn: "Inbox"}},
	}
	for _, op := range cases {
		t.Run(op.Type, func(t *testing.T) {
			data, err := json.Marshal(op)
			if err != nil {
				t.Fatalf("marshal %q: %v", op.Type, err)
			}
			var got board.MutationOp
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal %q: %v", op.Type, err)
			}
			if got.Type != op.Type {
				t.Errorf("type mismatch: want %q got %q", op.Type, got.Type)
			}
		})
	}
}

func TestMutationOpMarshalJSON_nilPayload(t *testing.T) {
	for _, typ := range board.MutationVariantNames() {
		t.Run(typ, func(t *testing.T) {
			op := board.MutationOp{Type: typ}
			if _, err := json.Marshal(op); err == nil {
				t.Errorf("want error marshaling %q with nil payload, got nil", typ)
			}
		})
	}
}

func TestMutationOpMarshalJSON_unknownType(t *testing.T) {
	op := board.MutationOp{Type: "bogus_op"}
	if _, err := json.Marshal(op); err == nil {
		t.Error("want error for unknown type, got nil")
	}
}

func TestMutationOpUnmarshalJSON_invalidJSON(t *testing.T) {
	var op board.MutationOp
	if err := json.Unmarshal([]byte("{invalid json"), &op); err == nil {
		t.Error("want error for invalid JSON, got nil")
	}
}

// TestRegistryCoversAllVariants asserts that every variant in the registry
// can round-trip Unmarshal → Marshal. Catches "added a struct field but
// forgot the registry entry" (Unmarshal fails) and "registry entry's set
// closure references the wrong field" (Marshal returns nil-payload error).
func TestRegistryCoversAllVariants(t *testing.T) {
	for _, typ := range board.MutationVariantNames() {
		t.Run(typ, func(t *testing.T) {
			raw := []byte(`{"type":"` + typ + `"}`)
			var op board.MutationOp
			if err := json.Unmarshal(raw, &op); err != nil {
				t.Fatalf("unmarshal %q: %v", typ, err)
			}
			if op.Type != typ {
				t.Errorf("type mismatch after unmarshal: want %q got %q", typ, op.Type)
			}
			if _, err := json.Marshal(op); err != nil {
				t.Errorf("marshal %q after unmarshal: %v", typ, err)
			}
		})
	}
}

// TestRegistryMatchesVectorSuite asserts that every Apply-driven variant has
// at least one vector in testdata/mutations/ exercising it. Replaces the
// parity runner's after-the-fact drift detection with prevention.
// move_card_to_board is exempt: its cross-board write is handler-driven, not
// Apply-driven, so it wouldn't be exercised by a vector even if one existed.
func TestRegistryMatchesVectorSuite(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "mutations")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read vectors dir: %v", err)
	}
	covered := make(map[string]bool)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, readErr := os.ReadFile(filepath.Join(root, e.Name()))
		if readErr != nil {
			t.Fatalf("read %s: %v", e.Name(), readErr)
		}
		var probe struct {
			Op struct {
				Type string `json:"type"`
			} `json:"op"`
		}
		if jerr := json.Unmarshal(raw, &probe); jerr != nil {
			t.Fatalf("parse %s: %v", e.Name(), jerr)
		}
		covered[probe.Op.Type] = true
	}
	for _, typ := range board.MutationVariantNames() {
		if typ == "move_card_to_board" {
			continue
		}
		if !covered[typ] {
			t.Errorf("variant %q is not exercised by any vector in testdata/mutations/", typ)
		}
	}
}
