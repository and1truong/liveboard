// Package workspace manages the liveboard workspace directory and board files.
package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/internal/writer"
	"github.com/and1truong/liveboard/pkg/models"
)

var defaultColumns = []string{"not now", "maybe?", "done"}

// Workspace manages boards in a directory.
type Workspace struct {
	Dir    string
	Engine *board.Engine
}

// Open returns a Workspace for the given directory.
func Open(dir string) *Workspace {
	return &Workspace{
		Dir:    dir,
		Engine: board.New(),
	}
}

// ListBoards returns all board files in the workspace.
func (w *Workspace) ListBoards() ([]models.Board, error) {
	entries, err := os.ReadDir(w.Dir)
	if err != nil {
		return nil, fmt.Errorf("read workspace: %w", err)
	}

	var boards []models.Board
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		// Skip non-board markdown files (like README).
		if entry.Name() == "README.md" {
			continue
		}
		path := filepath.Join(w.Dir, entry.Name())
		b, err := w.Engine.LoadBoard(path)
		if err != nil {
			continue // Skip unparseable files.
		}
		boards = append(boards, *b)
	}
	return boards, nil
}

// LoadBoard loads a board by name.
func (w *Workspace) LoadBoard(name string) (*models.Board, error) {
	path := w.BoardPath(name)
	return w.Engine.LoadBoard(path) //nolint:nilaway
}

// CreateBoard creates a new board with default columns.
func (w *Workspace) CreateBoard(name string) (*models.Board, error) {
	path := w.BoardPath(name)
	if _, err := os.Stat(path); err == nil {
		return nil, fmt.Errorf("board %q already exists", name)
	}

	cols := w.getDefaultColumns()
	b := &models.Board{
		Name:     name,
		FilePath: path,
	}
	for _, c := range cols {
		b.Columns = append(b.Columns, models.Column{Name: c})
	}

	content, err := writer.Render(b)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, err
	}
	return b, nil
}

// DeleteBoard removes a board file.
func (w *Workspace) DeleteBoard(name string) error {
	path := w.BoardPath(name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("board %q not found", name)
	}
	return os.Remove(path)
}

// BoardPath returns the file path for a board name.
func (w *Workspace) BoardPath(name string) string {
	return filepath.Join(w.Dir, name+".md")
}

// FindBoardByCardID searches all boards for a card with the given ID.
// Returns the board and file path.
func (w *Workspace) FindBoardByCardID(cardID string) (*models.Board, error) {
	boards, err := w.ListBoards()
	if err != nil {
		return nil, err
	}
	for _, b := range boards {
		for _, col := range b.Columns {
			for _, card := range col.Cards {
				if card.ID == cardID {
					return &b, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("card %s not found in any board", cardID)
}

func (w *Workspace) getDefaultColumns() []string {
	// Try settings.json (UI-configurable).
	settingsPath := filepath.Join(w.Dir, "settings.json")
	if data, err := os.ReadFile(settingsPath); err == nil {
		var s struct {
			DefaultColumns []string `json:"default_columns"`
		}
		if json.Unmarshal(data, &s) == nil && len(s.DefaultColumns) > 0 {
			return s.DefaultColumns
		}
	}
	// Try project config.
	configPath := filepath.Join(w.Dir, ".liveboard", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err == nil {
		var cfg models.Config
		if yaml.Unmarshal(data, &cfg) == nil && len(cfg.Board.DefaultColumns) > 0 {
			return cfg.Board.DefaultColumns
		}
	}
	return defaultColumns
}
