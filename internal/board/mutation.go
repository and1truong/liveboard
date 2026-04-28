package board

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/and1truong/liveboard/pkg/models"
)

// MutationOp is the tagged union of all board mutations.
// Exactly one of the pointer fields is populated based on Type.
// Kept in lockstep with MutationOp in the TS shell (P2).
type MutationOp struct {
	Type string `json:"type"`

	AddCard              *AddCardOp              `json:"-"`
	MoveCard             *MoveCardOp             `json:"-"`
	ReorderCard          *ReorderCardOp          `json:"-"`
	EditCard             *EditCardOp             `json:"-"`
	DeleteCard           *DeleteCardOp           `json:"-"`
	CompleteCard         *CompleteCardOp         `json:"-"`
	TagCard              *TagCardOp              `json:"-"`
	AddColumn            *AddColumnOp            `json:"-"`
	RenameColumn         *RenameColumnOp         `json:"-"`
	DeleteColumn         *DeleteColumnOp         `json:"-"`
	MoveColumn           *MoveColumnOp           `json:"-"`
	SortColumn           *SortColumnOp           `json:"-"`
	ToggleColumnCollapse *ToggleColumnCollapseOp `json:"-"`
	UpdateBoardMeta      *UpdateBoardMetaOp      `json:"-"`
	UpdateBoardMembers   *UpdateBoardMembersOp   `json:"-"`
	UpdateBoardIcon      *UpdateBoardIconOp      `json:"-"`
	UpdateBoardSettings  *UpdateBoardSettingsOp  `json:"-"`
	MoveCardToBoard      *MoveCardToBoardOp      `json:"-"`
}

// MoveCardToBoardOp are the params for a "move_card_to_board" mutation.
// This op crosses boards: the HTTP handler special-cases it to drive the
// atomic two-phase Engine.MoveCardToBoard. The Apply function here only
// removes the card from the source board (for optimistic/parity purposes).
type MoveCardToBoardOp struct {
	ColIdx    int    `json:"col_idx"`
	CardIdx   int    `json:"card_idx"`
	DstBoard  string `json:"dst_board"`
	DstColumn string `json:"dst_column"`
}

// AddCardOp are the params for an "add_card" mutation.
type AddCardOp struct {
	Column  string `json:"column"`
	Title   string `json:"title"`
	Prepend bool   `json:"prepend,omitempty"`
}

// MoveCardOp are the params for a "move_card" mutation.
type MoveCardOp struct {
	ColIdx       int    `json:"col_idx"`
	CardIdx      int    `json:"card_idx"`
	TargetColumn string `json:"target_column"`
}

// ReorderCardOp are the params for a "reorder_card" mutation.
type ReorderCardOp struct {
	ColIdx       int    `json:"col_idx"`
	CardIdx      int    `json:"card_idx"`
	BeforeIdx    int    `json:"before_idx"`
	TargetColumn string `json:"target_column"`
}

// EditCardOp are the params for an "edit_card" mutation.
type EditCardOp struct {
	ColIdx   int      `json:"col_idx"`
	CardIdx  int      `json:"card_idx"`
	Title    string   `json:"title"`
	Body     string   `json:"body"`
	Tags     []string `json:"tags"`
	Links    []string `json:"links"`
	Priority string   `json:"priority"`
	Due      string   `json:"due"`
	Assignee string   `json:"assignee"`
}

// DeleteCardOp are the params for a "delete_card" mutation.
type DeleteCardOp struct {
	ColIdx  int `json:"col_idx"`
	CardIdx int `json:"card_idx"`
}

// CompleteCardOp are the params for a "complete_card" mutation.
type CompleteCardOp struct {
	ColIdx  int `json:"col_idx"`
	CardIdx int `json:"card_idx"`
}

// TagCardOp are the params for a "tag_card" mutation.
type TagCardOp struct {
	ColIdx  int      `json:"col_idx"`
	CardIdx int      `json:"card_idx"`
	Tags    []string `json:"tags"`
}

// AddColumnOp are the params for an "add_column" mutation.
type AddColumnOp struct {
	Name string `json:"name"`
}

// RenameColumnOp are the params for a "rename_column" mutation.
type RenameColumnOp struct {
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`
}

// DeleteColumnOp are the params for a "delete_column" mutation.
type DeleteColumnOp struct {
	Name string `json:"name"`
}

// MoveColumnOp are the params for a "move_column" mutation.
type MoveColumnOp struct {
	Name     string `json:"name"`
	AfterCol string `json:"after_col"`
}

// SortColumnOp are the params for a "sort_column" mutation.
type SortColumnOp struct {
	ColIdx int    `json:"col_idx"`
	SortBy string `json:"sort_by"`
}

// ToggleColumnCollapseOp are the params for a "toggle_column_collapse" mutation.
type ToggleColumnCollapseOp struct {
	ColIdx int `json:"col_idx"`
}

// UpdateBoardMetaOp are the params for an "update_board_meta" mutation.
type UpdateBoardMetaOp struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// UpdateBoardMembersOp are the params for an "update_board_members" mutation.
type UpdateBoardMembersOp struct {
	Members []string `json:"members"`
}

// UpdateBoardIconOp are the params for an "update_board_icon" mutation.
// Pointer fields distinguish "leave unchanged" (nil) from "clear" ("") or set.
type UpdateBoardIconOp struct {
	Icon      *string `json:"icon,omitempty"`
	IconColor *string `json:"icon_color,omitempty"`
}

// UpdateBoardSettingsOp are the params for an "update_board_settings" mutation.
type UpdateBoardSettingsOp struct {
	Settings models.BoardSettings `json:"settings"`
}

// mustCast asserts v's dynamic type to T. The registry pairs each new() with
// its matching set/apply, so a failure here is an invariant violation in this
// file — panic is the right signal. Using the comma-ok form keeps errcheck
// (check-type-assertions) happy.
func mustCast[T any](v any) T {
	out, ok := v.(T)
	if !ok {
		var zero T
		panic(fmt.Sprintf("mutation registry: expected %T, got %T", zero, v))
	}
	return out
}

// variantSpec captures everything the dispatcher needs to know about one
// MutationOp variant: how to allocate a payload, how to read/write it on the
// union struct, and how to apply it to a board.
//
// Adding a new mutation = add a typed pointer field to MutationOp plus one
// entry to mutationRegistry. Marshal/Unmarshal/ApplyMutation all read from
// this map.
type variantSpec struct {
	new   func() any
	get   func(*MutationOp) (any, bool)
	set   func(*MutationOp, any)
	apply func(*models.Board, any) error
}

// mutationRegistry is the source of truth for every MutationOp variant.
// Coverage is asserted by TestRegistryCoversAllVariants and
// TestRegistryMatchesVectorSuite.
var mutationRegistry = map[string]variantSpec{
	"add_card": {
		new: func() any { return &AddCardOp{} },
		get: func(m *MutationOp) (any, bool) { return m.AddCard, m.AddCard != nil },
		set: func(m *MutationOp, v any) { m.AddCard = mustCast[*AddCardOp](v) },
		apply: func(b *models.Board, p any) error {
			op := mustCast[*AddCardOp](p)
			_, err := applyAddCard(b, op.Column, op.Title, op.Prepend)
			return err
		},
	},
	"move_card": {
		new: func() any { return &MoveCardOp{} },
		get: func(m *MutationOp) (any, bool) { return m.MoveCard, m.MoveCard != nil },
		set: func(m *MutationOp, v any) { m.MoveCard = mustCast[*MoveCardOp](v) },
		apply: func(b *models.Board, p any) error {
			op := mustCast[*MoveCardOp](p)
			return applyMoveCard(b, op.ColIdx, op.CardIdx, op.TargetColumn)
		},
	},
	"reorder_card": {
		new: func() any { return &ReorderCardOp{} },
		get: func(m *MutationOp) (any, bool) { return m.ReorderCard, m.ReorderCard != nil },
		set: func(m *MutationOp, v any) { m.ReorderCard = mustCast[*ReorderCardOp](v) },
		apply: func(b *models.Board, p any) error {
			op := mustCast[*ReorderCardOp](p)
			return applyReorderCard(b, op.ColIdx, op.CardIdx, op.BeforeIdx, op.TargetColumn)
		},
	},
	"edit_card": {
		new: func() any { return &EditCardOp{} },
		get: func(m *MutationOp) (any, bool) { return m.EditCard, m.EditCard != nil },
		set: func(m *MutationOp, v any) { m.EditCard = mustCast[*EditCardOp](v) },
		apply: func(b *models.Board, p any) error {
			op := mustCast[*EditCardOp](p)
			return applyEditCard(b, op.ColIdx, op.CardIdx, op.Title, op.Body, op.Tags, op.Links, op.Priority, op.Due, op.Assignee)
		},
	},
	"delete_card": {
		new: func() any { return &DeleteCardOp{} },
		get: func(m *MutationOp) (any, bool) { return m.DeleteCard, m.DeleteCard != nil },
		set: func(m *MutationOp, v any) { m.DeleteCard = mustCast[*DeleteCardOp](v) },
		apply: func(b *models.Board, p any) error {
			op := mustCast[*DeleteCardOp](p)
			return applyDeleteCard(b, op.ColIdx, op.CardIdx)
		},
	},
	"complete_card": {
		new: func() any { return &CompleteCardOp{} },
		get: func(m *MutationOp) (any, bool) { return m.CompleteCard, m.CompleteCard != nil },
		set: func(m *MutationOp, v any) { m.CompleteCard = mustCast[*CompleteCardOp](v) },
		apply: func(b *models.Board, p any) error {
			op := mustCast[*CompleteCardOp](p)
			return applyCompleteCard(b, op.ColIdx, op.CardIdx)
		},
	},
	"tag_card": {
		new: func() any { return &TagCardOp{} },
		get: func(m *MutationOp) (any, bool) { return m.TagCard, m.TagCard != nil },
		set: func(m *MutationOp, v any) { m.TagCard = mustCast[*TagCardOp](v) },
		apply: func(b *models.Board, p any) error {
			op := mustCast[*TagCardOp](p)
			return applyTagCard(b, op.ColIdx, op.CardIdx, op.Tags)
		},
	},
	"add_column": {
		new: func() any { return &AddColumnOp{} },
		get: func(m *MutationOp) (any, bool) { return m.AddColumn, m.AddColumn != nil },
		set: func(m *MutationOp, v any) { m.AddColumn = mustCast[*AddColumnOp](v) },
		apply: func(b *models.Board, p any) error {
			op := mustCast[*AddColumnOp](p)
			return applyAddColumn(b, op.Name)
		},
	},
	"rename_column": {
		new: func() any { return &RenameColumnOp{} },
		get: func(m *MutationOp) (any, bool) { return m.RenameColumn, m.RenameColumn != nil },
		set: func(m *MutationOp, v any) { m.RenameColumn = mustCast[*RenameColumnOp](v) },
		apply: func(b *models.Board, p any) error {
			op := mustCast[*RenameColumnOp](p)
			return applyRenameColumn(b, op.OldName, op.NewName)
		},
	},
	"delete_column": {
		new: func() any { return &DeleteColumnOp{} },
		get: func(m *MutationOp) (any, bool) { return m.DeleteColumn, m.DeleteColumn != nil },
		set: func(m *MutationOp, v any) { m.DeleteColumn = mustCast[*DeleteColumnOp](v) },
		apply: func(b *models.Board, p any) error {
			op := mustCast[*DeleteColumnOp](p)
			return applyDeleteColumn(b, op.Name)
		},
	},
	"move_column": {
		new: func() any { return &MoveColumnOp{} },
		get: func(m *MutationOp) (any, bool) { return m.MoveColumn, m.MoveColumn != nil },
		set: func(m *MutationOp, v any) { m.MoveColumn = mustCast[*MoveColumnOp](v) },
		apply: func(b *models.Board, p any) error {
			op := mustCast[*MoveColumnOp](p)
			return applyMoveColumn(b, op.Name, op.AfterCol)
		},
	},
	"sort_column": {
		new: func() any { return &SortColumnOp{} },
		get: func(m *MutationOp) (any, bool) { return m.SortColumn, m.SortColumn != nil },
		set: func(m *MutationOp, v any) { m.SortColumn = mustCast[*SortColumnOp](v) },
		apply: func(b *models.Board, p any) error {
			op := mustCast[*SortColumnOp](p)
			return applySortColumn(b, op.ColIdx, op.SortBy)
		},
	},
	"toggle_column_collapse": {
		new: func() any { return &ToggleColumnCollapseOp{} },
		get: func(m *MutationOp) (any, bool) { return m.ToggleColumnCollapse, m.ToggleColumnCollapse != nil },
		set: func(m *MutationOp, v any) { m.ToggleColumnCollapse = mustCast[*ToggleColumnCollapseOp](v) },
		apply: func(b *models.Board, p any) error {
			op := mustCast[*ToggleColumnCollapseOp](p)
			return applyToggleColumnCollapse(b, op.ColIdx)
		},
	},
	"update_board_meta": {
		new: func() any { return &UpdateBoardMetaOp{} },
		get: func(m *MutationOp) (any, bool) { return m.UpdateBoardMeta, m.UpdateBoardMeta != nil },
		set: func(m *MutationOp, v any) { m.UpdateBoardMeta = mustCast[*UpdateBoardMetaOp](v) },
		apply: func(b *models.Board, p any) error {
			op := mustCast[*UpdateBoardMetaOp](p)
			return applyUpdateBoardMeta(b, op.Name, op.Description)
		},
	},
	"update_board_members": {
		new: func() any { return &UpdateBoardMembersOp{} },
		get: func(m *MutationOp) (any, bool) { return m.UpdateBoardMembers, m.UpdateBoardMembers != nil },
		set: func(m *MutationOp, v any) { m.UpdateBoardMembers = mustCast[*UpdateBoardMembersOp](v) },
		apply: func(b *models.Board, p any) error {
			op := mustCast[*UpdateBoardMembersOp](p)
			return applyUpdateBoardMembers(b, op.Members)
		},
	},
	"update_board_icon": {
		new: func() any { return &UpdateBoardIconOp{} },
		get: func(m *MutationOp) (any, bool) { return m.UpdateBoardIcon, m.UpdateBoardIcon != nil },
		set: func(m *MutationOp, v any) { m.UpdateBoardIcon = mustCast[*UpdateBoardIconOp](v) },
		apply: func(b *models.Board, p any) error {
			op := mustCast[*UpdateBoardIconOp](p)
			return applyUpdateBoardIcon(b, op.Icon, op.IconColor)
		},
	},
	"update_board_settings": {
		new: func() any { return &UpdateBoardSettingsOp{} },
		get: func(m *MutationOp) (any, bool) { return m.UpdateBoardSettings, m.UpdateBoardSettings != nil },
		set: func(m *MutationOp, v any) { m.UpdateBoardSettings = mustCast[*UpdateBoardSettingsOp](v) },
		apply: func(b *models.Board, p any) error {
			op := mustCast[*UpdateBoardSettingsOp](p)
			return applyUpdateBoardSettings(b, op.Settings)
		},
	},
	"move_card_to_board": {
		new: func() any { return &MoveCardToBoardOp{} },
		get: func(m *MutationOp) (any, bool) { return m.MoveCardToBoard, m.MoveCardToBoard != nil },
		set: func(m *MutationOp, v any) { m.MoveCardToBoard = mustCast[*MoveCardToBoardOp](v) },
		// Source-side effect only: removes the card from the source board.
		// The cross-board atomic write is driven separately by the HTTP
		// handler via Engine.MoveCardToBoard (see handleMoveCardToBoard).
		apply: func(b *models.Board, p any) error {
			op := mustCast[*MoveCardToBoardOp](p)
			return applyDeleteCard(b, op.ColIdx, op.CardIdx)
		},
	},
}

// MutationVariantNames returns the canonical sorted list of mutation op
// type strings supported by this package. Source of truth: mutationRegistry.
// Exposed for tests that need to enumerate variants without hard-coding.
func MutationVariantNames() []string {
	names := make([]string, 0, len(mutationRegistry))
	for k := range mutationRegistry {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// RegisteredOpZeroValues returns a fresh zero-value pointer for each
// registered mutation variant, keyed by its type discriminator string.
// Used by reflection-driven consumers — primarily the cmd/gen-ts-mutations
// codegen that derives TypeScript types from the Go struct definitions.
func RegisteredOpZeroValues() map[string]any {
	out := make(map[string]any, len(mutationRegistry))
	for name, spec := range mutationRegistry {
		out[name] = spec.new()
	}
	return out
}

// MarshalJSON encodes the active variant merged with the "type" discriminator.
func (m MutationOp) MarshalJSON() ([]byte, error) {
	spec, ok := mutationRegistry[m.Type]
	if !ok {
		return nil, fmt.Errorf("unknown mutation op type: %q", m.Type)
	}
	variant, isSet := spec.get(&m)
	if !isSet {
		return nil, fmt.Errorf("MutationOp type=%q but payload is nil", m.Type)
	}
	variantBytes, err := json.Marshal(variant)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(variantBytes, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		fields = make(map[string]json.RawMessage)
	}
	typeBytes, _ := json.Marshal(m.Type)
	fields["type"] = json.RawMessage(typeBytes)
	return json.Marshal(fields)
}

// UnmarshalJSON decodes based on the `type` discriminator.
func (m *MutationOp) UnmarshalJSON(data []byte) error {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return err
	}
	m.Type = head.Type

	spec, ok := mutationRegistry[head.Type]
	if !ok {
		return fmt.Errorf("unknown mutation op type: %q", head.Type)
	}
	payload := spec.new()
	if err := json.Unmarshal(data, payload); err != nil {
		return err
	}
	spec.set(m, payload)
	return nil
}

// ApplyMutation mutates the board in-place according to op.
// Pure in-memory dispatcher — no disk IO, no locking, no version bump.
// HTTP handlers wrap this inside Engine.MutateBoard to add those concerns.
// Shared with the parity vector runner in internal/parity.
func ApplyMutation(b *models.Board, op MutationOp) error {
	spec, ok := mutationRegistry[op.Type]
	if !ok {
		return fmt.Errorf("unknown mutation op type: %q", op.Type)
	}
	payload, isSet := spec.get(&op)
	if !isSet {
		return fmt.Errorf("%s: missing params", op.Type)
	}
	return spec.apply(b, payload)
}
