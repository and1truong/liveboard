// Package web provides the HTMX-powered web UI for LiveBoard.
package web

import (
	"bytes"
	"html/template"
	"log"
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
)

// Handler manages web handlers and shared dependencies.
type Handler struct {
	ws               *workspace.Workspace
	eng              *board.Engine
	version          string
	ReadOnly         bool
	IsDesktop        bool
	SSE              *SSEBroker
	ReminderStore    *reminder.Store
	boardListTpl     *template.Template
	boardViewTpl     *template.Template
	boardGridTpl     *template.Template // partial: boards grid only
	boardContentTpl  *template.Template // partial: board content only
	sidebarBoardsTpl *template.Template // partial: sidebar board list
	reminderPageTpl  *template.Template // full: reminders page
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
			buf := mdBufPool.Get().(*bytes.Buffer)
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

// NewHandler creates a new web Handler.
func NewHandler(ws *workspace.Workspace, eng *board.Engine, version string, readOnly, isDesktop bool) *Handler {
	h := &Handler{
		ws:            ws,
		eng:           eng,
		version:       version,
		ReadOnly:      readOnly,
		IsDesktop:     isDesktop,
		SSE:           NewSSEBroker(),
		ReminderStore: reminder.NewStore(ws.Dir),
	}

	fm := funcMap()
	h.boardListTpl = template.Must(template.New("layout.html").Funcs(fm).ParseFS(tmplfs.FS, "layout.html", "board_list.html"))
	h.boardViewTpl = template.Must(template.New("layout.html").Funcs(fm).ParseFS(tmplfs.FS, "layout.html", "board_view.html"))
	h.reminderPageTpl = template.Must(template.New("layout.html").Funcs(fm).ParseFS(tmplfs.FS, "layout.html", "reminders.html"))

	// Partial templates for HTMX responses
	h.boardGridTpl = template.Must(template.New("boards-grid").Funcs(fm).ParseFS(tmplfs.FS, "board_list.html"))
	h.boardContentTpl = template.Must(template.New("board-content").Funcs(fm).ParseFS(tmplfs.FS, "board_view.html"))
	h.sidebarBoardsTpl = template.Must(template.New("sidebar-boards").Funcs(fm).ParseFS(tmplfs.FS, "layout.html"))

	return h
}

// renderFullPage renders a full page (layout + content) to the response writer.
func renderFullPage(w http.ResponseWriter, tpl *template.Template, model interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.Execute(w, model); err != nil {
		log.Printf("template render error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// renderPartial renders a named template block to the response writer.
func renderPartial(w http.ResponseWriter, tpl *template.Template, name string, model interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, name, model); err != nil {
		log.Printf("partial render error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// publishBoardEvent broadcasts a board update via SSE.
func (h *Handler) publishBoardEvent(slug string) {
	h.SSE.Publish(slug)
}
