// Package workspace manages the liveboard workspace directory and board files.
package workspace

import (
	"encoding/json"
	"fmt"
	"log"
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

// maxBoardIDDepth caps the directory depth at which boards are discovered.
// 1 = root-level boards only. 2 = root plus one subfolder level.
const maxBoardIDDepth = 2

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

// isSkippable filters workspace entries that are never boards or folders.
func isSkippable(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch name {
	case "README.md", "settings.json":
		return true
	}
	return false
}

// ListBoards returns all board files in the workspace (root + depth-1 folders).
func (w *Workspace) ListBoards() ([]models.Board, error) {
	var boards []models.Board
	err := w.walkBoards(func(relDir string, entry os.DirEntry) {
		path := filepath.Join(w.Dir, relDir, entry.Name())
		b, err := w.Engine.LoadBoard(path)
		if err != nil {
			log.Printf("workspace: skipping %s: %v", filepath.Join(relDir, entry.Name()), err)
			return
		}
		if fi, err := entry.Info(); err == nil {
			b.UpdatedAt = fi.ModTime()
			b.CreatedAt = fileBirthTime(fi)
		}
		boards = append(boards, *b)
	})
	if err != nil {
		return nil, err
	}
	return boards, nil
}

// ListBoardSummaries returns lightweight summaries without full card parsing.
func (w *Workspace) ListBoardSummaries() ([]parser.BoardSummaryInfo, error) {
	var summaries []parser.BoardSummaryInfo
	err := w.walkBoards(func(relDir string, entry os.DirEntry) {
		path := filepath.Join(w.Dir, relDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("workspace: skipping %s: %v", filepath.Join(relDir, entry.Name()), err)
			return
		}
		info, err := parser.ParseSummary(string(data))
		if err != nil {
			log.Printf("workspace: skipping %s: %v", filepath.Join(relDir, entry.Name()), err)
			return
		}
		info.Board.FilePath = path
		if fi, err := entry.Info(); err == nil {
			info.Board.UpdatedAt = fi.ModTime()
			info.Board.CreatedAt = fileBirthTime(fi)
		}
		summaries = append(summaries, *info)
	})
	if err != nil {
		return nil, err
	}
	return summaries, nil
}

// walkBoards visits every `.md` board file at root and depth 1. The visit
// callback receives the directory-relative path ("" for root) plus the entry.
// A read error on the root itself is returned; nested read errors are logged.
func (w *Workspace) walkBoards(visit func(relDir string, entry os.DirEntry)) error {
	entries, err := os.ReadDir(w.Dir)
	if err != nil {
		return fmt.Errorf("read workspace: %w", err)
	}
	for _, entry := range entries {
		if isSkippable(entry.Name()) {
			continue
		}
		if entry.IsDir() {
			w.visitSubdir(entry.Name(), visit)
			continue
		}
		if strings.HasSuffix(entry.Name(), ".md") {
			visit("", entry)
		}
	}
	return nil
}

// visitSubdir reads a workspace subdirectory and invokes visit for each
// .md file inside. Nested subdirectories are ignored (depth cap).
func (w *Workspace) visitSubdir(name string, visit func(relDir string, entry os.DirEntry)) {
	sub, err := os.ReadDir(filepath.Join(w.Dir, name))
	if err != nil {
		log.Printf("workspace: reading %s: %v", name, err)
		return
	}
	for _, entry := range sub {
		if entry.IsDir() || isSkippable(entry.Name()) {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".md") {
			visit(name, entry)
		}
	}
}

// LoadBoard loads a board by id (e.g. "roadmap" or "work/ideas").
func (w *Workspace) LoadBoard(id string) (*models.Board, error) {
	path, err := w.BoardPath(id)
	if err != nil {
		return nil, err
	}
	return w.Engine.LoadBoard(path) //nolint:nilaway
}

// CreateBoard creates a new board with default columns at the given id.
// The id is a relative path without the .md suffix; intermediate folders
// are created automatically.
func (w *Workspace) CreateBoard(id string) (*models.Board, error) {
	path, err := w.BoardPath(id)
	if err != nil {
		return nil, err
	}
	if _, statErr := os.Stat(path); statErr == nil {
		return nil, fmt.Errorf("board %q: %w", id, ErrAlreadyExists)
	}

	if mkErr := os.MkdirAll(filepath.Dir(path), 0755); mkErr != nil {
		return nil, mkErr
	}

	cols := w.getDefaultColumns()
	// Name is the last path segment (the file stem).
	name := id
	if i := strings.LastIndex(id, "/"); i >= 0 {
		name = id[i+1:]
	}
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

	if writeErr := os.WriteFile(path, []byte(content), 0644); writeErr != nil {
		return nil, writeErr
	}
	return b, nil
}

// DeleteBoard removes a board file. Empty parent folders are preserved.
func (w *Workspace) DeleteBoard(id string) error {
	path, err := w.BoardPath(id)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("board %q: %w", id, board.ErrNotFound)
	}
	return os.Remove(path)
}

// RenameBoard renames/moves a board from oldID to newID. Both ids may include
// a folder segment, so this is also the move-across-folders primitive.
// Errors: board.ErrNotFound if old missing, ErrAlreadyExists if new collides,
// ErrInvalidBoardName if either id is invalid.
func (w *Workspace) RenameBoard(oldID, newID string) (*models.Board, error) {
	newID = strings.TrimSpace(newID)
	oldPath, err := w.BoardPath(oldID)
	if err != nil {
		return nil, err
	}
	newPath, err := w.BoardPath(newID)
	if err != nil {
		return nil, err
	}
	if _, statErr := os.Stat(oldPath); os.IsNotExist(statErr) {
		return nil, fmt.Errorf("board %q: %w", oldID, board.ErrNotFound)
	}
	if oldPath != newPath {
		if _, statErr := os.Stat(newPath); statErr == nil {
			return nil, fmt.Errorf("board %q: %w", newID, ErrAlreadyExists)
		}
	}
	b, err := w.Engine.LoadBoard(oldPath)
	if err != nil {
		return nil, err
	}
	// Name is the new file stem, not the full id.
	newName := newID
	if i := strings.LastIndex(newID, "/"); i >= 0 {
		newName = newID[i+1:]
	}
	b.Name = newName
	b.FilePath = newPath
	b.Version++
	content, err := writer.Render(b)
	if err != nil {
		return nil, err
	}
	if oldPath != newPath {
		if err := os.MkdirAll(filepath.Dir(newPath), 0755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(newPath, []byte(content), 0644); err != nil {
			return nil, err
		}
		if err := os.Remove(oldPath); err != nil {
			return nil, err
		}
	} else {
		if err := os.WriteFile(oldPath, []byte(content), 0644); err != nil {
			return nil, err
		}
	}
	return b, nil
}

// ListFolders returns the names of direct subdirectories (non-hidden) that
// may contain boards. Empty folders are included.
func (w *Workspace) ListFolders() ([]string, error) {
	entries, err := os.ReadDir(w.Dir)
	if err != nil {
		return nil, fmt.Errorf("read workspace: %w", err)
	}
	var folders []string
	for _, entry := range entries {
		if !entry.IsDir() || isSkippable(entry.Name()) {
			continue
		}
		folders = append(folders, entry.Name())
	}
	return folders, nil
}

// CreateFolder creates a new folder under the workspace root. It fails if
// the name collides with an existing folder or with a root-level board file.
func (w *Workspace) CreateFolder(name string) error {
	if err := ValidateBoardName(name); err != nil {
		return err
	}
	path := filepath.Join(w.Dir, name)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("folder %q: %w", name, ErrAlreadyExists)
	}
	if _, err := os.Stat(path + ".md"); err == nil {
		return fmt.Errorf("folder %q: %w", name, ErrAlreadyExists)
	}
	return os.Mkdir(path, 0755)
}

// RenameFolder renames a folder. Callers are responsible for rewriting any
// pinned-board ids that referenced the old folder name.
func (w *Workspace) RenameFolder(oldName, newName string) error {
	if err := ValidateBoardName(oldName); err != nil {
		return err
	}
	if err := ValidateBoardName(newName); err != nil {
		return err
	}
	oldPath := filepath.Join(w.Dir, oldName)
	newPath := filepath.Join(w.Dir, newName)
	fi, err := os.Stat(oldPath)
	if os.IsNotExist(err) || (err == nil && !fi.IsDir()) {
		return fmt.Errorf("folder %q: %w", oldName, board.ErrNotFound)
	}
	if err != nil {
		return err
	}
	if oldPath == newPath {
		return nil
	}
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("folder %q: %w", newName, ErrAlreadyExists)
	}
	return os.Rename(oldPath, newPath)
}

// DeleteFolder removes an empty folder. Non-empty folders return an error.
func (w *Workspace) DeleteFolder(name string) error {
	if err := ValidateBoardName(name); err != nil {
		return err
	}
	path := filepath.Join(w.Dir, name)
	fi, err := os.Stat(path)
	if os.IsNotExist(err) || (err == nil && !fi.IsDir()) {
		return fmt.Errorf("folder %q: %w", name, board.ErrNotFound)
	}
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("folder %q: %w", name, ErrFolderNotEmpty)
	}
	return os.Remove(path)
}

// validBoardName allows alphanumeric, unicode letters, spaces, dashes, underscores, periods.
var validBoardName = regexp.MustCompile(`^[\p{L}\p{N} ._-]+$`)

// Sentinel errors for workspace operations.
var (
	// ErrInvalidBoardName is returned when a board name or id contains unsafe characters.
	ErrInvalidBoardName = fmt.Errorf("invalid board name")
	// ErrAlreadyExists is returned when trying to create a board or folder that already exists.
	ErrAlreadyExists = fmt.Errorf("already exists")
	// ErrFolderNotEmpty is returned when trying to delete a folder that still contains entries.
	ErrFolderNotEmpty = fmt.Errorf("folder not empty")
)

// ValidateBoardName checks that a single-segment name is safe for filenames.
// It rejects slashes, path traversal, and control characters.
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

// ValidateBoardID checks that a board id (possibly "folder/name") is safe.
// Each segment must pass ValidateBoardName and the id must have at most
// maxBoardIDDepth segments.
func ValidateBoardID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: empty id", ErrInvalidBoardName)
	}
	segs := strings.Split(id, "/")
	if len(segs) > maxBoardIDDepth {
		return fmt.Errorf("%w: too deep", ErrInvalidBoardName)
	}
	for _, seg := range segs {
		if seg == "." || seg == ".." {
			return fmt.Errorf("%w: path traversal", ErrInvalidBoardName)
		}
		if err := ValidateBoardName(seg); err != nil {
			return err
		}
	}
	return nil
}

// BoardPath returns the file path for a board id. The id may contain a
// single "/" to address a board inside a folder.
func (w *Workspace) BoardPath(id string) (string, error) {
	if err := ValidateBoardID(id); err != nil {
		return "", err
	}
	segs := strings.Split(id, "/")
	segs[len(segs)-1] += ".md"
	p := filepath.Join(append([]string{w.Dir}, segs...)...)
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
