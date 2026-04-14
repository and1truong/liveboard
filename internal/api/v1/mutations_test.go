package v1_test

import (
	"encoding/json"
	"testing"

	v1 "github.com/and1truong/liveboard/internal/api/v1"
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
