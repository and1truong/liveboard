// Package web provides the LiveView web UI for LiveBoard.
package web

import (
	"bytes"
	"context"
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"

	"github.com/jfyne/live"

	"github.com/and1truong/liveboard/internal/board"
	gitpkg "github.com/and1truong/liveboard/internal/git"
	"github.com/and1truong/liveboard/internal/workspace"
)

// Handler manages LiveView handlers and shared dependencies.
type Handler struct {
	ws           *workspace.Workspace
	eng          *board.Engine
	git          *gitpkg.Repository
	pubsub       *live.PubSub
	tmplDir      string
	boardListTpl *template.Template
	boardViewTpl *template.Template
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

	// Find template directory
	templateDirs := []string{
		filepath.Join("internal", "templates"),
		"templates",
		".",
	}

	for _, dir := range templateDirs {
		layoutPath := filepath.Join(dir, "layout.html")
		if _, err := template.ParseFiles(layoutPath); err == nil {
			h.tmplDir = dir
			break
		}
	}

	if h.tmplDir != "" {
		layoutFile := filepath.Join(h.tmplDir, "layout.html")
		h.boardListTpl = template.Must(template.ParseFiles(layoutFile, filepath.Join(h.tmplDir, "board_list.html")))
		h.boardViewTpl = template.Must(template.ParseFiles(layoutFile, filepath.Join(h.tmplDir, "board_view.html")))
	} else {
		// Empty templates for tests
		h.boardListTpl = template.New("empty")
		h.boardViewTpl = template.New("empty")
	}

	return h
}

// withAssignsRenderer is like live.WithTemplateRenderer but passes rc.Assigns
// (the model) directly to the template instead of the full RenderContext.
// This lets templates access model fields directly (e.g. {{.Title}}, {{.Board}}).
func withAssignsRenderer(t *template.Template) live.HandlerConfig {
	return func(h *live.Handler) error {
		h.RenderHandler = func(ctx context.Context, rc *live.RenderContext) (io.Reader, error) {
			var buf bytes.Buffer
			if err := t.Execute(&buf, rc.Assigns); err != nil {
				return nil, err
			}
			return &buf, nil
		}
		return nil
	}
}

// BoardListHandler returns an http.Handler for the board list page.
func (h *Handler) BoardListHandler() http.Handler {
	boardListHandler := live.NewHandler(
		withAssignsRenderer(h.boardListTpl),
	)
	boardListHandler.MountHandler = h.mountBoardList
	boardListHandler.HandleEvent("create-board", h.handleCreateBoard)
	boardListHandler.HandleEvent("delete-board", h.handleDeleteBoard)
	boardListHandler.HandleEvent("set-board-icon", h.handleSetBoardIconList)
	boardListHandler.HandleParams(h.handleParams)

	return live.NewHttpHandler(context.Background(), boardListHandler,
		live.WithSocketStateStore(live.NewMemorySocketStateStore(context.Background())),
	)
}

// BoardViewHandler returns an http.Handler for a single board view.
func (h *Handler) BoardViewHandler() http.Handler {
	boardViewHandler := live.NewHandler(
		withAssignsRenderer(h.boardViewTpl),
	)
	boardViewHandler.MountHandler = h.mountBoardView
	boardViewHandler.HandleEvent("create-card", h.handleCreateCard)
	boardViewHandler.HandleEvent("move-card", h.handleMoveCard)
	boardViewHandler.HandleEvent("reorder-card", h.handleReorderCard)
	boardViewHandler.HandleEvent("delete-card", h.handleDeleteCard)
	boardViewHandler.HandleEvent("toggle-complete", h.handleToggleComplete)
	boardViewHandler.HandleEvent("create-column", h.handleCreateColumn)
	boardViewHandler.HandleEvent("edit-card", h.handleEditCard)
	boardViewHandler.HandleEvent("rename-column", h.handleRenameColumn)
	boardViewHandler.HandleEvent("delete-column", h.handleDeleteColumn)
	boardViewHandler.HandleEvent("update-board-meta", h.handleUpdateBoardMeta)
	boardViewHandler.HandleEvent("toggle-column-collapse", h.handleToggleColumnCollapse)
	boardViewHandler.HandleEvent("sort-column", h.handleSortColumn)
	boardViewHandler.HandleEvent("update-board-settings", h.handleUpdateBoardSettings)
	boardViewHandler.HandleEvent("set-board-icon", h.handleSetBoardIcon)
	boardViewHandler.HandleSelf("board_update", h.handleBoardUpdate)

	return live.NewHttpHandler(context.Background(), boardViewHandler,
		live.WithSocketStateStore(live.NewMemorySocketStateStore(context.Background())),
	)
}

// publishBoardEvent broadcasts a board update to all subscribers.
func (h *Handler) publishBoardEvent(boardName string) {
	ctx := context.Background()
	_ = h.pubsub.Publish(ctx, "board_update", live.Event{
		T:        "board_update",
		SelfData: boardName,
	})
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
