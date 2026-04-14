package v1

import (
	"encoding/json"
	"fmt"

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
}

type AddCardOp struct {
	Column  string `json:"column"`
	Title   string `json:"title"`
	Prepend bool   `json:"prepend,omitempty"`
}

type MoveCardOp struct {
	ColIdx       int    `json:"col_idx"`
	CardIdx      int    `json:"card_idx"`
	TargetColumn string `json:"target_column"`
}

type ReorderCardOp struct {
	ColIdx       int    `json:"col_idx"`
	CardIdx      int    `json:"card_idx"`
	BeforeIdx    int    `json:"before_idx"`
	TargetColumn string `json:"target_column"`
}

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

type DeleteCardOp struct {
	ColIdx  int `json:"col_idx"`
	CardIdx int `json:"card_idx"`
}

type CompleteCardOp struct {
	ColIdx  int `json:"col_idx"`
	CardIdx int `json:"card_idx"`
}

type TagCardOp struct {
	ColIdx  int      `json:"col_idx"`
	CardIdx int      `json:"card_idx"`
	Tags    []string `json:"tags"`
}

type AddColumnOp struct {
	Name string `json:"name"`
}

type RenameColumnOp struct {
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`
}

type DeleteColumnOp struct {
	Name string `json:"name"`
}

type MoveColumnOp struct {
	Name     string `json:"name"`
	AfterCol string `json:"after_col"`
}

type SortColumnOp struct {
	ColIdx int    `json:"col_idx"`
	SortBy string `json:"sort_by"`
}

type ToggleColumnCollapseOp struct {
	ColIdx int `json:"col_idx"`
}

type UpdateBoardMetaOp struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

type UpdateBoardMembersOp struct {
	Members []string `json:"members"`
}

type UpdateBoardIconOp struct {
	Icon string `json:"icon"`
}

type UpdateBoardSettingsOp struct {
	Settings models.BoardSettings `json:"settings"`
}

// MarshalJSON encodes the active variant merged with the "type" discriminator.
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
