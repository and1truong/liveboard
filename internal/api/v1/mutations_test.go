package v1_test

import (
	"encoding/json"
	"testing"

	v1 "github.com/and1truong/liveboard/internal/api/v1"
	"github.com/and1truong/liveboard/pkg/models"
)

func TestMutationOpUnmarshalAddCard(t *testing.T) {
	raw := []byte(`{"type":"add_card","column":"Todo","title":"hello","prepend":false}`)
	var op v1.MutationOp
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
	var op v1.MutationOp
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
	var op v1.MutationOp
	if err := json.Unmarshal(raw, &op); err == nil {
		t.Fatal("want error for unknown op type")
	}
}

func TestMutationOpUnmarshalAllVariants(t *testing.T) {
	cases := []struct {
		name        string
		jsonStr     string
		wantType    string
		nonNilField func(*v1.MutationOp) bool
	}{
		{
			name:        "add_card",
			jsonStr:     `{"type":"add_card","column":"Todo","title":"x"}`,
			wantType:    "add_card",
			nonNilField: func(m *v1.MutationOp) bool { return m.AddCard != nil },
		},
		{
			name:        "move_card",
			jsonStr:     `{"type":"move_card","col_idx":0,"card_idx":1,"target_column":"Done"}`,
			wantType:    "move_card",
			nonNilField: func(m *v1.MutationOp) bool { return m.MoveCard != nil },
		},
		{
			name:        "reorder_card",
			jsonStr:     `{"type":"reorder_card","col_idx":0,"card_idx":1,"before_idx":0,"target_column":"Todo"}`,
			wantType:    "reorder_card",
			nonNilField: func(m *v1.MutationOp) bool { return m.ReorderCard != nil },
		},
		{
			name:        "edit_card",
			jsonStr:     `{"type":"edit_card","col_idx":0,"card_idx":0,"title":"t","body":"b","tags":[],"priority":"high","due":"","assignee":""}`,
			wantType:    "edit_card",
			nonNilField: func(m *v1.MutationOp) bool { return m.EditCard != nil },
		},
		{
			name:        "delete_card",
			jsonStr:     `{"type":"delete_card","col_idx":0,"card_idx":0}`,
			wantType:    "delete_card",
			nonNilField: func(m *v1.MutationOp) bool { return m.DeleteCard != nil },
		},
		{
			name:        "complete_card",
			jsonStr:     `{"type":"complete_card","col_idx":0,"card_idx":0}`,
			wantType:    "complete_card",
			nonNilField: func(m *v1.MutationOp) bool { return m.CompleteCard != nil },
		},
		{
			name:        "tag_card",
			jsonStr:     `{"type":"tag_card","col_idx":0,"card_idx":0,"tags":["x"]}`,
			wantType:    "tag_card",
			nonNilField: func(m *v1.MutationOp) bool { return m.TagCard != nil },
		},
		{
			name:        "add_column",
			jsonStr:     `{"type":"add_column","name":"Backlog"}`,
			wantType:    "add_column",
			nonNilField: func(m *v1.MutationOp) bool { return m.AddColumn != nil },
		},
		{
			name:        "rename_column",
			jsonStr:     `{"type":"rename_column","old_name":"Todo","new_name":"Doing"}`,
			wantType:    "rename_column",
			nonNilField: func(m *v1.MutationOp) bool { return m.RenameColumn != nil },
		},
		{
			name:        "delete_column",
			jsonStr:     `{"type":"delete_column","name":"Done"}`,
			wantType:    "delete_column",
			nonNilField: func(m *v1.MutationOp) bool { return m.DeleteColumn != nil },
		},
		{
			name:        "move_column",
			jsonStr:     `{"type":"move_column","name":"Done","after_col":"Todo"}`,
			wantType:    "move_column",
			nonNilField: func(m *v1.MutationOp) bool { return m.MoveColumn != nil },
		},
		{
			name:        "sort_column",
			jsonStr:     `{"type":"sort_column","col_idx":0,"sort_by":"priority"}`,
			wantType:    "sort_column",
			nonNilField: func(m *v1.MutationOp) bool { return m.SortColumn != nil },
		},
		{
			name:        "toggle_column_collapse",
			jsonStr:     `{"type":"toggle_column_collapse","col_idx":2}`,
			wantType:    "toggle_column_collapse",
			nonNilField: func(m *v1.MutationOp) bool { return m.ToggleColumnCollapse != nil },
		},
		{
			name:        "update_board_meta",
			jsonStr:     `{"type":"update_board_meta","name":"X","description":"","tags":[]}`,
			wantType:    "update_board_meta",
			nonNilField: func(m *v1.MutationOp) bool { return m.UpdateBoardMeta != nil },
		},
		{
			name:        "update_board_members",
			jsonStr:     `{"type":"update_board_members","members":["alice"]}`,
			wantType:    "update_board_members",
			nonNilField: func(m *v1.MutationOp) bool { return m.UpdateBoardMembers != nil },
		},
		{
			name:        "update_board_icon",
			jsonStr:     `{"type":"update_board_icon","icon":"🚀"}`,
			wantType:    "update_board_icon",
			nonNilField: func(m *v1.MutationOp) bool { return m.UpdateBoardIcon != nil },
		},
		{
			name:        "update_board_settings",
			jsonStr:     `{"type":"update_board_settings","settings":{}}`,
			wantType:    "update_board_settings",
			nonNilField: func(m *v1.MutationOp) bool { return m.UpdateBoardSettings != nil },
		},
		{
			name:        "update_tag_colors",
			jsonStr:     `{"type":"update_tag_colors","tag_colors":{"go":"#00ff00"}}`,
			wantType:    "update_tag_colors",
			nonNilField: func(m *v1.MutationOp) bool { return m.UpdateTagColors != nil },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var op v1.MutationOp
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
		op   v1.MutationOp
	}{
		{
			name: "add_card",
			op: v1.MutationOp{
				Type:    "add_card",
				AddCard: &v1.AddCardOp{Column: "Todo", Title: "my card", Prepend: true},
			},
		},
		{
			name: "move_card",
			op: v1.MutationOp{
				Type:     "move_card",
				MoveCard: &v1.MoveCardOp{ColIdx: 1, CardIdx: 2, TargetColumn: "Done"},
			},
		},
		{
			name: "edit_card",
			op: v1.MutationOp{
				Type: "edit_card",
				EditCard: &v1.EditCardOp{
					ColIdx: 0, CardIdx: 3,
					Title: "edited", Body: "body text",
					Tags: []string{"go", "api"}, Priority: "high",
					Due: "2026-12-31", Assignee: "alice",
				},
			},
		},
		{
			name: "update_board_settings",
			op: v1.MutationOp{
				Type: "update_board_settings",
				UpdateBoardSettings: &v1.UpdateBoardSettingsOp{
					Settings: models.BoardSettings{ShowCheckbox: boolPtr(true)},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// marshal
			data, err := json.Marshal(tc.op)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			// unmarshal back
			var got v1.MutationOp
			if jerr := json.Unmarshal(data, &got); jerr != nil {
				t.Fatalf("unmarshal: %v", jerr)
			}
			if got.Type != tc.op.Type {
				t.Errorf("type mismatch: want %q got %q", tc.op.Type, got.Type)
			}
			// marshal again and compare JSON bytes
			data2, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("re-marshal: %v", err)
			}
			// compare as maps to be order-agnostic
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

func boolPtr(b bool) *bool { return &b }
