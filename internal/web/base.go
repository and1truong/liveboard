package web

import (
	"html/template"
	"net/http"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/internal/workspace"
)

// Base holds shared dependencies embedded by all sub-handlers.
type Base struct {
	ws        *workspace.Workspace
	eng       *board.Engine
	version   string
	ReadOnly  bool
	IsDesktop bool
	SSE       *SSEBroker
}

// publishBoardEvent broadcasts a board update via SSE.
func (b *Base) publishBoardEvent(slug string) {
	b.SSE.Publish(slug)
}

// renderFullPage renders a full page (layout + content) to the response writer.
func renderFullPage(w http.ResponseWriter, tpl *template.Template, model interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.Execute(w, model); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// renderPartial renders a named template block to the response writer.
func renderPartial(w http.ResponseWriter, tpl *template.Template, name string, model interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, name, model); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
