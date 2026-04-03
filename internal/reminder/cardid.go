package reminder

import "github.com/and1truong/liveboard/pkg/models"

// EnsureCardIDs assigns a ULID to each card that doesn't have one.
// Returns true if any IDs were added (indicating the board should be saved).
func EnsureCardIDs(board *models.Board) bool {
	changed := false
	for ci := range board.Columns {
		for ri := range board.Columns[ci].Cards {
			card := &board.Columns[ci].Cards[ri]
			if card.Metadata == nil {
				card.Metadata = map[string]string{}
			}
			if card.Metadata["id"] == "" {
				card.Metadata["id"] = GenerateID()
				changed = true
			}
		}
	}
	return changed
}

// GetCardID returns the ULID of a card, or empty string if not set.
func GetCardID(card *models.Card) string {
	if card.Metadata == nil {
		return ""
	}
	return card.Metadata["id"]
}
