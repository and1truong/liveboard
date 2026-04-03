// Package web provides the HTMX-powered web UI for LiveBoard.
package web

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"
	"sync"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/internal/reminder"
	tmplfs "github.com/and1truong/liveboard/internal/templates"
	"github.com/and1truong/liveboard/internal/workspace"
	"github.com/and1truong/liveboard/pkg/models"
)

// Handler is the coordinator that owns all sub-handlers and shared dependencies.
type Handler struct {
	*Base
	BoardList *BoardListHandler
	BoardView *BoardViewHandler
	Reminders *ReminderHandler
	Settings  *SettingsHandler
}

// mdBufPool reuses buffers for markdown rendering to avoid per-call allocation.
var mdBufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

// mdRenderer is a goldmark instance configured for safe HTML output.
var mdRenderer = goldmark.New(
	goldmark.WithExtensions(extension.Linkify),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
		html.WithXHTML(),
	),
)

// funcMap returns the template function map shared by all templates.
func funcMap() template.FuncMap {
	return template.FuncMap{
		"cardID": func(m map[string]string) string {
			if m == nil {
				return ""
			}
			return m["id"]
		},
		"md": func(s string) template.HTML {
			buf, _ := mdBufPool.Get().(*bytes.Buffer)
			buf.Reset()
			defer mdBufPool.Put(buf)
			if err := mdRenderer.Convert([]byte(s), buf); err != nil {
				return template.HTML(template.HTMLEscapeString(s))
			}
			out := strings.ReplaceAll(buf.String(), "<a href=", `<a target="_blank" rel="noopener" href=`)
			return template.HTML(out) //nolint:gosec // goldmark output, raw HTML disabled by default
		},
	}
}

// NewHandler creates a new web Handler with all sub-handlers wired together.
func NewHandler(ws *workspace.Workspace, eng *board.Engine, version string, readOnly, isDesktop bool) *Handler {
	base := &Base{
		ws:        ws,
		eng:       eng,
		version:   version,
		ReadOnly:  readOnly,
		IsDesktop: isDesktop,
		SSE:       NewSSEBroker(),
	}

	fm := funcMap()

	reminderStore := reminder.NewStore(ws.Dir)

	bl := &BoardListHandler{
		Base:             base,
		boardListTpl:     template.Must(template.New("layout.html").Funcs(fm).ParseFS(tmplfs.FS, "layout.html", "board_list.html")),
		boardGridTpl:     template.Must(template.New("boards-grid").Funcs(fm).ParseFS(tmplfs.FS, "board_list.html")),
		sidebarBoardsTpl: template.Must(template.New("sidebar-boards").Funcs(fm).ParseFS(tmplfs.FS, "layout.html")),
	}

	bv := &BoardViewHandler{
		Base:            base,
		boardViewTpl:    template.Must(template.New("layout.html").Funcs(fm).ParseFS(tmplfs.FS, "layout.html", "board_view.html")),
		boardContentTpl: template.Must(template.New("board-content").Funcs(fm).ParseFS(tmplfs.FS, "board_view.html")),
	}

	rem := &ReminderHandler{
		Base:            base,
		Store:           reminderStore,
		reminderPageTpl: template.Must(template.New("layout.html").Funcs(fm).ParseFS(tmplfs.FS, "layout.html", "reminders.html")),
	}

	settings := &SettingsHandler{Base: base}

	// Wire reminder callbacks so board handlers don't import reminder package for side-effects.
	bv.onCardCompleted = func(slug, cardID string) {
		_ = reminderStore.RemoveByCardID(slug, cardID)
	}
	bv.onDueDateChanged = func(slug, cardID, due string) {
		s := LoadSettingsFromDir(ws.Dir)
		tz := s.ReminderTimezone
		if tz == "" {
			tz = "Local"
		}
		_ = reminderStore.RecalculateRelativeReminder(slug, cardID, due, tz)
	}

	return &Handler{
		Base:      base,
		BoardList: bl,
		BoardView: bv,
		Reminders: rem,
		Settings:  settings,
	}
}

// ReminderStore returns the reminder store for external access (e.g. scheduler).
func (h *Handler) ReminderStore() *reminder.Store {
	return h.Reminders.Store
}

// ExportHandler returns an HTTP handler that exports the workspace as a static ZIP.
func (h *Handler) ExportHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		exportHandler(h.Base)(w, r)
	})
}

// --- Forwarding helpers for tests that call internal methods ---

func (h *Handler) boardListModel() (BoardListModel, error) {
	return h.BoardList.boardListModel()
}
func (h *Handler) boardViewModel(slug string) (BoardViewModel, error) {
	return h.BoardView.boardViewModel(slug)
}
func (h *Handler) mutateBoard(slug string, clientVersion int, op func(*models.Board) error) (BoardViewModel, error) {
	return h.BoardView.mutateBoard(slug, clientVersion, op)
}
func (h *Handler) loadSettings() AppSettings {
	return LoadSettingsFromDir(h.ws.Dir)
}
func (h *Handler) saveSettings(s AppSettings) error {
	return saveSettingsToDir(h.ws.Dir, s)
}
func (h *Handler) layoutSettings(s AppSettings) LayoutSettings {
	return h.Base.layoutSettings(s)
}
func (h *Handler) handleConflict(w http.ResponseWriter, slug string) {
	h.BoardView.handleConflict(w, slug)
}
