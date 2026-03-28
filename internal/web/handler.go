// Package web provides the HTMX-powered web UI for LiveBoard.
package web

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"

	"github.com/and1truong/liveboard/internal/board"
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
	boardListTpl     *template.Template
	boardViewTpl     *template.Template
	boardGridTpl     *template.Template // partial: boards grid only
	boardContentTpl  *template.Template // partial: board content only
	sidebarBoardsTpl *template.Template // partial: sidebar board list
}

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
		"md": func(s string) template.HTML {
			var buf bytes.Buffer
			if err := mdRenderer.Convert([]byte(s), &buf); err != nil {
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
		ws:        ws,
		eng:       eng,
		version:   version,
		ReadOnly:  readOnly,
		IsDesktop: isDesktop,
		SSE:       NewSSEBroker(),
	}

	fm := funcMap()
	h.boardListTpl = template.Must(template.New("layout.html").Funcs(fm).ParseFS(tmplfs.FS, "layout.html", "board_list.html"))
	h.boardViewTpl = template.Must(template.New("layout.html").Funcs(fm).ParseFS(tmplfs.FS, "layout.html", "board_view.html"))

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
