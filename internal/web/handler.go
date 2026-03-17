// Package web provides the LiveView web UI for LiveBoard.
package web

import (
	"context"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/jfyne/live"

	"github.com/and1truong/liveboard/internal/board"
	gitpkg "github.com/and1truong/liveboard/internal/git"
	"github.com/and1truong/liveboard/internal/workspace"
)

// Handler manages LiveView handlers and shared dependencies.
type Handler struct {
	ws     *workspace.Workspace
	eng    *board.Engine
	git    *gitpkg.Repository
	pubsub *live.PubSub
	tmpl   *template.Template
}

// NewHandler creates a new web Handler.
func NewHandler(ws *workspace.Workspace, eng *board.Engine, git *gitpkg.Repository) *Handler {
	h := &Handler{
		ws:  ws,
		eng: eng,
		git: git,
	}

	// Create PubSub for real-time updates
	ctx := context.Background()
	h.pubsub = live.NewPubSub(ctx, live.NewLocalTransport())

	// Create template engine - try multiple paths
	templatePatterns := []string{
		filepath.Join("internal", "templates", "*.html"),
		filepath.Join("templates", "*.html"),
		"*.html",
	}

	var err error
	for _, pattern := range templatePatterns {
		h.tmpl, err = template.ParseGlob(pattern)
		if err == nil && h.tmpl != nil {
			break
		}
	}

	// If no templates found, create an empty template (for tests)
	if h.tmpl == nil {
		h.tmpl = template.New("empty")
	}

	return h
}

// BoardListHandler returns an http.Handler for the board list page.
func (h *Handler) BoardListHandler() http.Handler {
	boardListHandler := live.NewHandler(
		live.WithTemplateRenderer(h.tmpl),
	)
	boardListHandler.MountHandler = h.mountBoardList
	boardListHandler.HandleEvent("create-board", h.handleCreateBoard)
	boardListHandler.HandleEvent("delete-board", h.handleDeleteBoard)
	boardListHandler.HandleEvent("show-create-form", h.handleShowCreateForm)
	boardListHandler.HandleEvent("cancel-create", h.handleCancelCreate)
	boardListHandler.HandleParams(h.handleParams)

	return live.NewHttpHandler(context.Background(), boardListHandler,
		live.WithSocketStateStore(live.NewMemorySocketStateStore(context.Background())),
	)
}

// BoardViewHandler returns an http.Handler for a single board view.
func (h *Handler) BoardViewHandler() http.Handler {
	boardViewHandler := live.NewHandler(
		live.WithTemplateRenderer(h.tmpl),
	)
	boardViewHandler.MountHandler = h.mountBoardView
	boardViewHandler.HandleEvent("create-card", h.handleCreateCard)
	boardViewHandler.HandleEvent("move-card", h.handleMoveCard)
	boardViewHandler.HandleEvent("delete-card", h.handleDeleteCard)
	boardViewHandler.HandleEvent("toggle-complete", h.handleToggleComplete)
	boardViewHandler.HandleEvent("create-column", h.handleCreateColumn)
	boardViewHandler.HandleEvent("show-add-card", h.handleShowAddCard)
	boardViewHandler.HandleEvent("cancel-add-card", h.handleCancelAddCard)
	boardViewHandler.HandleSelf("board_update", h.handleBoardUpdate)

	return live.NewHttpHandler(context.Background(), boardViewHandler,
		live.WithSocketStateStore(live.NewMemorySocketStateStore(context.Background())),
	)
}

// publishBoardEvent broadcasts a board update to all subscribers.
func (h *Handler) publishBoardEvent(boardName string, _ string) {
	ctx := context.Background()
	_ = h.pubsub.Publish(ctx, "board_update", live.Event{
		T:        "board_update",
		SelfData: boardName,
	})
}
