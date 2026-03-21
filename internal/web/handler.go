// Package web provides the HTMX-powered web UI for LiveBoard.
package web

import (
	"html/template"
	"log"
	"net/http"

	"github.com/and1truong/liveboard/internal/board"
	gitpkg "github.com/and1truong/liveboard/internal/git"
	tmplfs "github.com/and1truong/liveboard/internal/templates"
	"github.com/and1truong/liveboard/internal/workspace"
)

// Handler manages web handlers and shared dependencies.
type Handler struct {
	ws              *workspace.Workspace
	eng             *board.Engine
	git             *gitpkg.Repository
	SSE             *SSEBroker
	boardListTpl    *template.Template
	boardViewTpl    *template.Template
	boardGridTpl    *template.Template // partial: boards grid only
	boardContentTpl *template.Template // partial: board content only
}

// NewHandler creates a new web Handler.
func NewHandler(ws *workspace.Workspace, eng *board.Engine, git *gitpkg.Repository) *Handler {
	h := &Handler{
		ws:  ws,
		eng: eng,
		git: git,
		SSE: NewSSEBroker(),
	}

	h.boardListTpl = template.Must(template.ParseFS(tmplfs.FS, "layout.html", "board_list.html"))
	h.boardViewTpl = template.Must(template.ParseFS(tmplfs.FS, "layout.html", "board_view.html"))

	// Partial templates for HTMX responses
	h.boardGridTpl = template.Must(template.New("boards-grid").ParseFS(tmplfs.FS, "board_list.html"))
	h.boardContentTpl = template.Must(template.New("board-content").ParseFS(tmplfs.FS, "board_view.html"))

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

// commitWithHandling performs a git commit and logs any errors.
func (h *Handler) commitWithHandling(boardPath, msg string) {
	if h.git == nil {
		return
	}
	if err := h.git.Commit(boardPath, msg); err != nil {
		log.Printf("git commit failed for %s: %v", boardPath, err)
	}
}

// commitRemoveWithHandling performs a git commit for removal and logs any errors.
func (h *Handler) commitRemoveWithHandling(boardPath, msg string) {
	if h.git == nil {
		return
	}
	if err := h.git.CommitRemove(boardPath, msg); err != nil {
		log.Printf("git commit remove failed for %s: %v", boardPath, err)
	}
}
