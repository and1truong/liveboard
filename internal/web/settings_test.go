package web

import (
	"testing"

	"github.com/and1truong/liveboard/pkg/models"
)

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func TestSanitizeBoardSettings_dropsInvalidEnums(t *testing.T) {
	bs := models.BoardSettings{
		ShowCheckbox:    boolPtr(false),     // bool — passes through
		CardPosition:    strPtr("sideways"), // invalid → nil
		ExpandColumns:   boolPtr(true),      // bool — passes through
		ViewMode:        strPtr("garbage"),  // invalid → nil
		CardDisplayMode: strPtr("normal"),   // intentionally not validated — passes through
		WeekStart:       strPtr("funday"),   // invalid → nil
	}
	SanitizeBoardSettings(&bs)

	if bs.ShowCheckbox == nil || *bs.ShowCheckbox != false {
		t.Errorf("ShowCheckbox should pass through unchanged")
	}
	if bs.CardPosition != nil {
		t.Errorf("invalid CardPosition should be cleared, got %q", *bs.CardPosition)
	}
	if bs.ExpandColumns == nil || *bs.ExpandColumns != true {
		t.Errorf("ExpandColumns should pass through unchanged")
	}
	if bs.ViewMode != nil {
		t.Errorf("invalid ViewMode should be cleared, got %q", *bs.ViewMode)
	}
	if bs.CardDisplayMode == nil || *bs.CardDisplayMode != "normal" {
		t.Errorf("CardDisplayMode should pass through (not validated), got %v", bs.CardDisplayMode)
	}
	if bs.WeekStart != nil {
		t.Errorf("invalid WeekStart should be cleared, got %q", *bs.WeekStart)
	}
}

func TestSanitizeBoardSettings_validValuesPreserved(t *testing.T) {
	cases := []struct {
		name string
		bs   models.BoardSettings
	}{
		{"card_position append", models.BoardSettings{CardPosition: strPtr("append")}},
		{"card_position prepend", models.BoardSettings{CardPosition: strPtr("prepend")}},
		{"view_mode board", models.BoardSettings{ViewMode: strPtr("board")}},
		{"view_mode list", models.BoardSettings{ViewMode: strPtr("list")}},
		{"view_mode calendar", models.BoardSettings{ViewMode: strPtr("calendar")}},
		{"view_mode table", models.BoardSettings{ViewMode: strPtr("table")}},
		{"week_start sunday", models.BoardSettings{WeekStart: strPtr("sunday")}},
		{"week_start monday", models.BoardSettings{WeekStart: strPtr("monday")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := tc.bs
			SanitizeBoardSettings(&tc.bs)
			if !pointersEqualBS(before, tc.bs) {
				t.Errorf("valid value mutated: before=%+v after=%+v", before, tc.bs)
			}
		})
	}
}

func TestSanitizeBoardSettings_nilFieldsLeftAlone(t *testing.T) {
	bs := models.BoardSettings{}
	SanitizeBoardSettings(&bs)
	if bs.ShowCheckbox != nil || bs.CardPosition != nil || bs.ExpandColumns != nil ||
		bs.ViewMode != nil || bs.CardDisplayMode != nil || bs.WeekStart != nil {
		t.Errorf("nil patch should remain nil, got %+v", bs)
	}
}

func pointersEqualBS(a, b models.BoardSettings) bool {
	eqBool := func(x, y *bool) bool { return (x == nil) == (y == nil) && (x == nil || *x == *y) }
	eqStr := func(x, y *string) bool { return (x == nil) == (y == nil) && (x == nil || *x == *y) }
	return eqBool(a.ShowCheckbox, b.ShowCheckbox) &&
		eqStr(a.CardPosition, b.CardPosition) &&
		eqBool(a.ExpandColumns, b.ExpandColumns) &&
		eqStr(a.ViewMode, b.ViewMode) &&
		eqStr(a.CardDisplayMode, b.CardDisplayMode) &&
		eqStr(a.WeekStart, b.WeekStart)
}
