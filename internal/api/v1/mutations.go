package v1

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/pkg/models"
)

type mutationRequest struct {
	ClientVersion int        `json:"client_version"`
	Op            MutationOp `json:"op"`
}

func (d Deps) postMutation(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	slug := chi.URLParam(r, "slug")

	boardPath, pathErr := d.Workspace.BoardPath(slug)
	if pathErr != nil {
		writeError(w, pathErr)
		return
	}

	var req mutationRequest
	if decodeErr := decodeJSON(r, &req); decodeErr != nil {
		writeError(w, fmt.Errorf("%w: %v", errInvalid, decodeErr))
		return
	}

	updated, dispatchErr := Dispatch(d.Engine, boardPath, req.ClientVersion, req.Op)
	if dispatchErr != nil {
		writeError(w, dispatchErr)
		return
	}

	if d.SSE != nil {
		d.SSE.Publish(slug)
	}

	_ = json.NewEncoder(w).Encode(updated)
}

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
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// UpdateBoardMembersOp are the params for an "update_board_members" mutation.
type UpdateBoardMembersOp struct {
	Members []string `json:"members"`
}

// UpdateBoardIconOp are the params for an "update_board_icon" mutation.
type UpdateBoardIconOp struct {
	Icon string `json:"icon"`
}

// UpdateBoardSettingsOp are the params for an "update_board_settings" mutation.
type UpdateBoardSettingsOp struct {
	Settings models.BoardSettings `json:"settings"`
}

// MarshalJSON encodes the active variant merged with the "type" discriminator.
//
//nolint:cyclop,gocognit,funlen // switch over 17 mutation variants — inherently large
func (m MutationOp) MarshalJSON() ([]byte, error) {
	var variant any
	switch m.Type {
	case "add_card":
		if m.AddCard == nil {
			return nil, fmt.Errorf("MutationOp type=%q but AddCard is nil", m.Type)
		}
		variant = m.AddCard
	case "move_card":
		if m.MoveCard == nil {
			return nil, fmt.Errorf("MutationOp type=%q but MoveCard is nil", m.Type)
		}
		variant = m.MoveCard
	case "reorder_card":
		if m.ReorderCard == nil {
			return nil, fmt.Errorf("MutationOp type=%q but ReorderCard is nil", m.Type)
		}
		variant = m.ReorderCard
	case "edit_card":
		if m.EditCard == nil {
			return nil, fmt.Errorf("MutationOp type=%q but EditCard is nil", m.Type)
		}
		variant = m.EditCard
	case "delete_card":
		if m.DeleteCard == nil {
			return nil, fmt.Errorf("MutationOp type=%q but DeleteCard is nil", m.Type)
		}
		variant = m.DeleteCard
	case "complete_card":
		if m.CompleteCard == nil {
			return nil, fmt.Errorf("MutationOp type=%q but CompleteCard is nil", m.Type)
		}
		variant = m.CompleteCard
	case "tag_card":
		if m.TagCard == nil {
			return nil, fmt.Errorf("MutationOp type=%q but TagCard is nil", m.Type)
		}
		variant = m.TagCard
	case "add_column":
		if m.AddColumn == nil {
			return nil, fmt.Errorf("MutationOp type=%q but AddColumn is nil", m.Type)
		}
		variant = m.AddColumn
	case "rename_column":
		if m.RenameColumn == nil {
			return nil, fmt.Errorf("MutationOp type=%q but RenameColumn is nil", m.Type)
		}
		variant = m.RenameColumn
	case "delete_column":
		if m.DeleteColumn == nil {
			return nil, fmt.Errorf("MutationOp type=%q but DeleteColumn is nil", m.Type)
		}
		variant = m.DeleteColumn
	case "move_column":
		if m.MoveColumn == nil {
			return nil, fmt.Errorf("MutationOp type=%q but MoveColumn is nil", m.Type)
		}
		variant = m.MoveColumn
	case "sort_column":
		if m.SortColumn == nil {
			return nil, fmt.Errorf("MutationOp type=%q but SortColumn is nil", m.Type)
		}
		variant = m.SortColumn
	case "toggle_column_collapse":
		if m.ToggleColumnCollapse == nil {
			return nil, fmt.Errorf("MutationOp type=%q but ToggleColumnCollapse is nil", m.Type)
		}
		variant = m.ToggleColumnCollapse
	case "update_board_meta":
		if m.UpdateBoardMeta == nil {
			return nil, fmt.Errorf("MutationOp type=%q but UpdateBoardMeta is nil", m.Type)
		}
		variant = m.UpdateBoardMeta
	case "update_board_members":
		if m.UpdateBoardMembers == nil {
			return nil, fmt.Errorf("MutationOp type=%q but UpdateBoardMembers is nil", m.Type)
		}
		variant = m.UpdateBoardMembers
	case "update_board_icon":
		if m.UpdateBoardIcon == nil {
			return nil, fmt.Errorf("MutationOp type=%q but UpdateBoardIcon is nil", m.Type)
		}
		variant = m.UpdateBoardIcon
	case "update_board_settings":
		if m.UpdateBoardSettings == nil {
			return nil, fmt.Errorf("MutationOp type=%q but UpdateBoardSettings is nil", m.Type)
		}
		variant = m.UpdateBoardSettings
	default:
		return nil, fmt.Errorf("unknown mutation op type: %q", m.Type)
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
//
//nolint:cyclop,funlen // switch over 17 mutation variants — inherently large
func (m *MutationOp) UnmarshalJSON(data []byte) error {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return err
	}
	m.Type = head.Type

	switch head.Type {
	case "add_card":
		m.AddCard = &AddCardOp{}
		return json.Unmarshal(data, m.AddCard)
	case "move_card":
		m.MoveCard = &MoveCardOp{}
		return json.Unmarshal(data, m.MoveCard)
	case "reorder_card":
		m.ReorderCard = &ReorderCardOp{}
		return json.Unmarshal(data, m.ReorderCard)
	case "edit_card":
		m.EditCard = &EditCardOp{}
		return json.Unmarshal(data, m.EditCard)
	case "delete_card":
		m.DeleteCard = &DeleteCardOp{}
		return json.Unmarshal(data, m.DeleteCard)
	case "complete_card":
		m.CompleteCard = &CompleteCardOp{}
		return json.Unmarshal(data, m.CompleteCard)
	case "tag_card":
		m.TagCard = &TagCardOp{}
		return json.Unmarshal(data, m.TagCard)
	case "add_column":
		m.AddColumn = &AddColumnOp{}
		return json.Unmarshal(data, m.AddColumn)
	case "rename_column":
		m.RenameColumn = &RenameColumnOp{}
		return json.Unmarshal(data, m.RenameColumn)
	case "delete_column":
		m.DeleteColumn = &DeleteColumnOp{}
		return json.Unmarshal(data, m.DeleteColumn)
	case "move_column":
		m.MoveColumn = &MoveColumnOp{}
		return json.Unmarshal(data, m.MoveColumn)
	case "sort_column":
		m.SortColumn = &SortColumnOp{}
		return json.Unmarshal(data, m.SortColumn)
	case "toggle_column_collapse":
		m.ToggleColumnCollapse = &ToggleColumnCollapseOp{}
		return json.Unmarshal(data, m.ToggleColumnCollapse)
	case "update_board_meta":
		m.UpdateBoardMeta = &UpdateBoardMetaOp{}
		return json.Unmarshal(data, m.UpdateBoardMeta)
	case "update_board_members":
		m.UpdateBoardMembers = &UpdateBoardMembersOp{}
		return json.Unmarshal(data, m.UpdateBoardMembers)
	case "update_board_icon":
		m.UpdateBoardIcon = &UpdateBoardIconOp{}
		return json.Unmarshal(data, m.UpdateBoardIcon)
	case "update_board_settings":
		m.UpdateBoardSettings = &UpdateBoardSettingsOp{}
		return json.Unmarshal(data, m.UpdateBoardSettings)
	default:
		return fmt.Errorf("unknown mutation op type: %q", head.Type)
	}
}

// Dispatch executes a MutationOp against the engine using a single MutateBoard
// call so clientVersion is checked atomically with the mutation.
// It returns the mutated board (with version already incremented in-place) so
// callers can encode it directly without a second LoadBoard.
func Dispatch(eng *board.Engine, boardPath string, clientVersion int, op MutationOp) (*models.Board, error) {
	var out *models.Board
	err := eng.MutateBoard(boardPath, clientVersion, func(b *models.Board) error {
		if e := applyOp(b, op); e != nil {
			return e
		}
		out = b
		return nil
	})
	return out, err
}

// applyOp mutates the in-memory board according to op.
// It calls the exported board.Apply* functions so logic is never duplicated.
//
//nolint:cyclop,gocognit,funlen // switch over 17 mutation variants — inherently large
func applyOp(b *models.Board, op MutationOp) error {
	switch op.Type {
	case "add_card":
		if op.AddCard == nil {
			return fmt.Errorf("add_card: missing params")
		}
		_, err := board.ApplyAddCard(b, op.AddCard.Column, op.AddCard.Title, op.AddCard.Prepend)
		return err
	case "move_card":
		if op.MoveCard == nil {
			return fmt.Errorf("move_card: missing params")
		}
		return board.ApplyMoveCard(b, op.MoveCard.ColIdx, op.MoveCard.CardIdx, op.MoveCard.TargetColumn)
	case "reorder_card":
		if op.ReorderCard == nil {
			return fmt.Errorf("reorder_card: missing params")
		}
		p := op.ReorderCard
		return board.ApplyReorderCard(b, p.ColIdx, p.CardIdx, p.BeforeIdx, p.TargetColumn)
	case "edit_card":
		if op.EditCard == nil {
			return fmt.Errorf("edit_card: missing params")
		}
		p := op.EditCard
		return board.ApplyEditCard(b, p.ColIdx, p.CardIdx, p.Title, p.Body, p.Tags, p.Priority, p.Due, p.Assignee)
	case "delete_card":
		if op.DeleteCard == nil {
			return fmt.Errorf("delete_card: missing params")
		}
		return board.ApplyDeleteCard(b, op.DeleteCard.ColIdx, op.DeleteCard.CardIdx)
	case "complete_card":
		if op.CompleteCard == nil {
			return fmt.Errorf("complete_card: missing params")
		}
		return board.ApplyCompleteCard(b, op.CompleteCard.ColIdx, op.CompleteCard.CardIdx)
	case "tag_card":
		if op.TagCard == nil {
			return fmt.Errorf("tag_card: missing params")
		}
		return board.ApplyTagCard(b, op.TagCard.ColIdx, op.TagCard.CardIdx, op.TagCard.Tags)
	case "add_column":
		if op.AddColumn == nil {
			return fmt.Errorf("add_column: missing params")
		}
		return board.ApplyAddColumn(b, op.AddColumn.Name)
	case "rename_column":
		if op.RenameColumn == nil {
			return fmt.Errorf("rename_column: missing params")
		}
		return board.ApplyRenameColumn(b, op.RenameColumn.OldName, op.RenameColumn.NewName)
	case "delete_column":
		if op.DeleteColumn == nil {
			return fmt.Errorf("delete_column: missing params")
		}
		return board.ApplyDeleteColumn(b, op.DeleteColumn.Name)
	case "move_column":
		if op.MoveColumn == nil {
			return fmt.Errorf("move_column: missing params")
		}
		return board.ApplyMoveColumn(b, op.MoveColumn.Name, op.MoveColumn.AfterCol)
	case "sort_column":
		if op.SortColumn == nil {
			return fmt.Errorf("sort_column: missing params")
		}
		return board.ApplySortColumn(b, op.SortColumn.ColIdx, op.SortColumn.SortBy)
	case "toggle_column_collapse":
		if op.ToggleColumnCollapse == nil {
			return fmt.Errorf("toggle_column_collapse: missing params")
		}
		return board.ApplyToggleColumnCollapse(b, op.ToggleColumnCollapse.ColIdx)
	case "update_board_meta":
		if op.UpdateBoardMeta == nil {
			return fmt.Errorf("update_board_meta: missing params")
		}
		p := op.UpdateBoardMeta
		return board.ApplyUpdateBoardMeta(b, p.Name, p.Description, p.Tags)
	case "update_board_members":
		if op.UpdateBoardMembers == nil {
			return fmt.Errorf("update_board_members: missing params")
		}
		return board.ApplyUpdateBoardMembers(b, op.UpdateBoardMembers.Members)
	case "update_board_icon":
		if op.UpdateBoardIcon == nil {
			return fmt.Errorf("update_board_icon: missing params")
		}
		return board.ApplyUpdateBoardIcon(b, op.UpdateBoardIcon.Icon)
	case "update_board_settings":
		if op.UpdateBoardSettings == nil {
			return fmt.Errorf("update_board_settings: missing params")
		}
		return board.ApplyUpdateBoardSettings(b, op.UpdateBoardSettings.Settings)
	default:
		return fmt.Errorf("unknown mutation op type: %q", op.Type)
	}
}
