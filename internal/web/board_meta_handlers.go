package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/pkg/models"
)

// HandleUpdateBoardMeta handles POST /board/{slug}/meta.
func (bv *BoardViewHandler) HandleUpdateBoardMeta(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		bv.renderBoardContent(w, model)
		return
	}

	name := r.FormValue("board_name")
	description := r.FormValue("description")
	tagsRaw := r.FormValue("tags")
	tagColorsRaw := r.FormValue("tag_colors")

	parts := strings.Split(tagsRaw, ",")
	tags := make([]string, 0, len(parts))
	for _, t := range parts {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}

	version := formVersion(r)
	model, mutErr := bv.mutateBoard(slug, version, func(b *models.Board) error {
		if name != "" {
			b.Name = name
		}
		b.Description = description
		b.Tags = tags
		if tagColorsRaw != "" {
			var tagColors map[string]string
			if err := json.Unmarshal([]byte(tagColorsRaw), &tagColors); err == nil {
				if len(tagColors) == 0 {
					b.TagColors = nil
				} else {
					b.TagColors = tagColors
				}
			}
		}
		return nil
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		bv.handleConflict(w, slug)
		return
	}
	bv.renderBoardContent(w, model)
}

// parseBoardSettingsForm extracts board settings from form values.
func parseBoardSettingsForm(r *http.Request) models.BoardSettings {
	var s models.BoardSettings
	if v := r.FormValue("show_checkbox"); v != "" {
		b := v == "true"
		s.ShowCheckbox = &b
	}
	if v := r.FormValue("card_position"); v == "prepend" || v == "append" {
		s.CardPosition = &v
	}
	if v := r.FormValue("expand_columns"); v != "" {
		b := v == "true"
		s.ExpandColumns = &b
	}
	if v := r.FormValue("view_mode"); v == "board" || v == "list" || v == "table" || v == "calendar" {
		s.ViewMode = &v
	}
	if v := r.FormValue("week_start"); v == "sunday" || v == "monday" {
		s.WeekStart = &v
	}
	if v := r.FormValue("card_display_mode"); v == "full" || v == "hide" || v == "trim" {
		s.CardDisplayMode = &v
	}
	return s
}

// HandleUpdateBoardSettings handles POST /board/{slug}/settings.
func (bv *BoardViewHandler) HandleUpdateBoardSettings(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		bv.renderBoardContent(w, model)
		return
	}

	settings := parseBoardSettingsForm(r)

	version := formVersion(r)
	model, mutErr := bv.mutateBoard(slug, version, func(b *models.Board) error {
		b.Settings = settings
		return nil
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		bv.handleConflict(w, slug)
		return
	}
	bv.renderBoardContent(w, model)
}

// HandleSetBoardIcon handles POST /board/{slug}/icon.
func (bv *BoardViewHandler) HandleSetBoardIcon(w http.ResponseWriter, r *http.Request) {
	slug := slugFromRequest(r)
	if slug == "" {
		model := BoardViewModel{Error: "Board name is required"}
		bv.renderBoardContent(w, model)
		return
	}

	icon := r.FormValue("icon")

	version := formVersion(r)
	model, mutErr := bv.mutateBoard(slug, version, func(b *models.Board) error {
		b.Icon = icon
		return nil
	})
	if errors.Is(mutErr, board.ErrVersionConflict) {
		bv.handleConflict(w, slug)
		return
	}
	bv.renderBoardContent(w, model)
}
