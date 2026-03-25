// Package workspace manages the liveboard workspace directory and board files.
package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/internal/parser"
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
		if fi, err := entry.Info(); err == nil {
			b.UpdatedAt = fi.ModTime()
			b.CreatedAt = fileBirthTime(fi)
		}
		boards = append(boards, *b)
	}
	return boards, nil
}

// ListBoardSummaries returns lightweight summaries without full card parsing.
func (w *Workspace) ListBoardSummaries() ([]parser.BoardSummaryInfo, error) {
	entries, err := os.ReadDir(w.Dir)
	if err != nil {
		return nil, fmt.Errorf("read workspace: %w", err)
	}

	var summaries []parser.BoardSummaryInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") || entry.Name() == "README.md" {
			continue
		}
		path := filepath.Join(w.Dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		info, err := parser.ParseSummary(string(data))
		if err != nil {
			continue
		}
		info.Board.FilePath = path
		if fi, err := entry.Info(); err == nil {
			info.Board.UpdatedAt = fi.ModTime()
			info.Board.CreatedAt = fileBirthTime(fi)
		}
		summaries = append(summaries, *info)
	}
	return summaries, nil
}

// LoadBoard loads a board by name.
func (w *Workspace) LoadBoard(name string) (*models.Board, error) {
	path, err := w.BoardPath(name)
	if err != nil {
		return nil, err
	}
	return w.Engine.LoadBoard(path) //nolint:nilaway
}

// CreateBoard creates a new board with default columns.
func (w *Workspace) CreateBoard(name string) (*models.Board, error) {
	path, err := w.BoardPath(name)
	if err != nil {
		return nil, err
	}
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
	path, err := w.BoardPath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("board %q not found", name)
	}
	return os.Remove(path)
}

// validBoardName allows alphanumeric, unicode letters, spaces, dashes, underscores, periods.
var validBoardName = regexp.MustCompile(`^[\p{L}\p{N} ._-]+$`)

// ErrInvalidBoardName is returned when a board name contains unsafe characters.
var ErrInvalidBoardName = fmt.Errorf("invalid board name")

// ValidateBoardName checks that a board name is safe for use as a filename.
func ValidateBoardName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: empty name", ErrInvalidBoardName)
	}
	if !validBoardName.MatchString(name) {
		return fmt.Errorf("%w: contains unsafe characters", ErrInvalidBoardName)
	}
	if !filepath.IsLocal(name + ".md") {
		return fmt.Errorf("%w: path traversal", ErrInvalidBoardName)
	}
	return nil
}

// BoardPath returns the file path for a board name.
func (w *Workspace) BoardPath(name string) (string, error) {
	if err := ValidateBoardName(name); err != nil {
		return "", err
	}
	p := filepath.Join(w.Dir, name+".md")
	// Belt-and-suspenders: ensure resolved path is inside workspace.
	if rel, err := filepath.Rel(w.Dir, p); err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("%w: path escapes workspace", ErrInvalidBoardName)
	}
	return p, nil
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
