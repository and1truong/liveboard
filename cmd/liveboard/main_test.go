package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/internal/defaults"
	"github.com/and1truong/liveboard/internal/workspace"
	"github.com/and1truong/liveboard/pkg/models"
)

const cliTestBoard = `---
name: CLI Test Board
---

## Backlog

- [ ] Task one

## In Progress

## Done
`

// setupCLI initializes the global CLI state used by all commands and returns
// the temp workspace directory.
func setupCLI(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	ws = workspace.Open(dir)
	eng = board.New()
	return dir
}

// createCLIBoard writes a test board file into dir and returns its path.
func createCLIBoard(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name+".md")
	if err := os.WriteFile(path, []byte(cliTestBoard), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// suppressStdout redirects os.Stdout to /dev/null for the duration of the test
// so command output doesn't pollute test logs.
func suppressStdout(t *testing.T) {
	t.Helper()
	old := os.Stdout
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = devNull
	t.Cleanup(func() {
		_ = devNull.Close()
		os.Stdout = old
	})
}

func TestColumnNames(t *testing.T) {
	b := &models.Board{
		Columns: []models.Column{
			{Name: "Backlog"},
			{Name: "In Progress"},
			{Name: "Done"},
		},
	}
	got := columnNames(b)
	want := "Backlog, In Progress, Done"
	if got != want {
		t.Errorf("columnNames = %q, want %q", got, want)
	}
}

func TestColumnNames_Single(t *testing.T) {
	b := &models.Board{
		Columns: []models.Column{{Name: "Backlog"}},
	}
	got := columnNames(b)
	if got != "Backlog" {
		t.Errorf("columnNames = %q, want %q", got, "Backlog")
	}
}

func TestColumnNames_Empty(t *testing.T) {
	b := &models.Board{}
	got := columnNames(b)
	if got != "" {
		t.Errorf("columnNames = %q, want %q", got, "")
	}
}

// --- Board command tests ---

func TestBoardListCmd_NoBoards(t *testing.T) {
	suppressStdout(t)
	setupCLI(t)

	cmd := boardListCmd()
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestBoardListCmd_WithBoards(t *testing.T) {
	suppressStdout(t)
	dir := setupCLI(t)
	createCLIBoard(t, dir, "roadmap")

	cmd := boardListCmd()
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestBoardCreateCmd(t *testing.T) {
	suppressStdout(t)
	dir := setupCLI(t)

	cmd := boardCreateCmd()
	if err := cmd.RunE(cmd, []string{"myboard"}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "myboard.md")); os.IsNotExist(err) {
		t.Error("board file not created")
	}
}

func TestBoardCreateCmd_DuplicateReturnsError(t *testing.T) {
	suppressStdout(t)
	dir := setupCLI(t)
	createCLIBoard(t, dir, "myboard")

	cmd := boardCreateCmd()
	if err := cmd.RunE(cmd, []string{"myboard"}); err == nil {
		t.Error("expected error when creating duplicate board")
	}
}

func TestBoardDeleteCmd(t *testing.T) {
	suppressStdout(t)
	dir := setupCLI(t)
	createCLIBoard(t, dir, "myboard")

	cmd := boardDeleteCmd()
	if err := cmd.RunE(cmd, []string{"myboard"}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "myboard.md")); !os.IsNotExist(err) {
		t.Error("board file should be deleted")
	}
}

func TestBoardDeleteCmd_NotFoundReturnsError(t *testing.T) {
	suppressStdout(t)
	setupCLI(t)

	cmd := boardDeleteCmd()
	if err := cmd.RunE(cmd, []string{"nonexistent"}); err == nil {
		t.Error("expected error for missing board")
	}
}

// --- Card command tests ---

func TestCardAddCmd(t *testing.T) {
	suppressStdout(t)
	dir := setupCLI(t)
	createCLIBoard(t, dir, "myboard")

	cmd := cardAddCmd()
	if err := cmd.RunE(cmd, []string{"myboard", "New Task"}); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "myboard.md"))
	if !strings.Contains(string(data), "New Task") {
		t.Error("card not found in board file")
	}
}

func TestCardAddCmd_WithColumnFlag(t *testing.T) {
	suppressStdout(t)
	dir := setupCLI(t)
	createCLIBoard(t, dir, "myboard")

	cmd := cardAddCmd()
	if err := cmd.Flags().Set("column", "In Progress"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, []string{"myboard", "In Progress Task"}); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "myboard.md"))
	if !strings.Contains(string(data), "In Progress Task") {
		t.Error("card not found in board file")
	}
}

func TestCardMoveCmd(t *testing.T) {
	suppressStdout(t)
	dir := setupCLI(t)
	createCLIBoard(t, dir, "myboard")

	// Move card at col=0, card=0 to "In Progress"
	cmd := cardMoveCmd()
	if err := cmd.RunE(cmd, []string{"myboard", "0", "0", "In Progress"}); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "myboard.md"))
	content := string(data)
	// Card should now be under "In Progress".
	inProgressIdx := strings.Index(content, "## In Progress")
	cardIdx := strings.Index(content, "Task one")
	if inProgressIdx == -1 || cardIdx == -1 || cardIdx < inProgressIdx {
		t.Error("card not found under In Progress after move")
	}
}

func TestCardMoveCmd_InvalidIndexReturnsError(t *testing.T) {
	suppressStdout(t)
	dir := setupCLI(t)
	createCLIBoard(t, dir, "myboard")

	cmd := cardMoveCmd()
	if err := cmd.RunE(cmd, []string{"myboard", "99", "0", "Done"}); err == nil {
		t.Error("expected error for out-of-range column index")
	}
}

func TestCardCompleteCmd(t *testing.T) {
	suppressStdout(t)
	dir := setupCLI(t)
	createCLIBoard(t, dir, "myboard")

	cmd := cardCompleteCmd()
	if err := cmd.RunE(cmd, []string{"myboard", "0", "0"}); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "myboard.md"))
	if !strings.Contains(string(data), "- [x] Task one") {
		t.Error("card not marked as completed")
	}
}

func TestCardTagCmd(t *testing.T) {
	suppressStdout(t)
	dir := setupCLI(t)
	createCLIBoard(t, dir, "myboard")

	cmd := cardTagCmd()
	if err := cmd.RunE(cmd, []string{"myboard", "0", "0", "urgent", "backend"}); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "myboard.md"))
	if !strings.Contains(string(data), "urgent") {
		t.Error("tag 'urgent' not found in board file")
	}
}

func TestCardDeleteCmd(t *testing.T) {
	suppressStdout(t)
	dir := setupCLI(t)
	createCLIBoard(t, dir, "myboard")

	cmd := cardDeleteCmd()
	if err := cmd.RunE(cmd, []string{"myboard", "0", "0"}); err != nil {
		t.Fatal(err)
	}

	b, err := ws.LoadBoard("myboard")
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Columns[0].Cards) != 0 {
		t.Error("deleted card still present")
	}
}

func TestCardShowCmd(t *testing.T) {
	suppressStdout(t)
	dir := setupCLI(t)
	createCLIBoard(t, dir, "myboard")

	cmd := cardShowCmd()
	if err := cmd.RunE(cmd, []string{"myboard", "0", "0"}); err != nil {
		t.Fatal(err)
	}
}

// --- Column command tests ---

func TestColumnAddCmd(t *testing.T) {
	suppressStdout(t)
	dir := setupCLI(t)
	createCLIBoard(t, dir, "myboard")

	cmd := columnAddCmd()
	if err := cmd.RunE(cmd, []string{"myboard", "Testing"}); err != nil {
		t.Fatal(err)
	}

	b, err := ws.LoadBoard("myboard")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, col := range b.Columns {
		if col.Name == "Testing" {
			found = true
		}
	}
	if !found {
		t.Error("column 'Testing' not found after add")
	}
}

func TestColumnDeleteCmd(t *testing.T) {
	suppressStdout(t)
	dir := setupCLI(t)
	createCLIBoard(t, dir, "myboard")

	cmd := columnDeleteCmd()
	if err := cmd.RunE(cmd, []string{"myboard", "Done"}); err != nil {
		t.Fatal(err)
	}

	b, err := ws.LoadBoard("myboard")
	if err != nil {
		t.Fatal(err)
	}
	for _, col := range b.Columns {
		if col.Name == "Done" {
			t.Error("column 'Done' still present after delete")
		}
	}
}

func TestColumnMoveCmd(t *testing.T) {
	suppressStdout(t)
	dir := setupCLI(t)
	createCLIBoard(t, dir, "myboard")

	cmd := columnMoveCmd()
	if err := cmd.Flags().Set("after", "Backlog"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, []string{"myboard", "Done"}); err != nil {
		t.Fatal(err)
	}

	b, err := ws.LoadBoard("myboard")
	if err != nil {
		t.Fatal(err)
	}
	// After moving Done after Backlog, order should be: Backlog, Done, In Progress.
	if len(b.Columns) < 2 {
		t.Fatalf("expected at least 2 columns, got %d", len(b.Columns))
	}
	if b.Columns[0].Name != "Backlog" {
		t.Errorf("column[0] = %q, want %q", b.Columns[0].Name, "Backlog")
	}
	if b.Columns[1].Name != "Done" {
		t.Errorf("column[1] = %q, want %q", b.Columns[1].Name, "Done")
	}
}

// --- Export command tests ---

func TestExportCmd(t *testing.T) {
	dir := setupCLI(t)
	createCLIBoard(t, dir, "export-test")
	suppressStdout(t)

	outDir := filepath.Join(t.TempDir(), "exported")
	cmd := exportCmd()
	cmd.SetArgs([]string{"--output", outDir})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Error("expected exported files")
	}
}

func TestExportCmd_DefaultOutput(t *testing.T) {
	dir := setupCLI(t)
	createCLIBoard(t, dir, "test")
	suppressStdout(t)

	old, _ := os.Getwd()
	_ = os.Chdir(t.TempDir())
	t.Cleanup(func() { _ = os.Chdir(old) })

	cmd := exportCmd()
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

// --- Serve/MCP command construction tests ---

func TestServeCmdFlags(t *testing.T) {
	cmd := serveCmd()
	if cmd.Use != "serve" {
		t.Errorf("Use = %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("missing Short description")
	}
	if f := cmd.Flags().Lookup("port"); f == nil {
		t.Error("missing --port flag")
	}
	if f := cmd.Flags().Lookup("host"); f == nil {
		t.Error("missing --host flag")
	}
	if f := cmd.Flags().Lookup("readonly"); f == nil {
		t.Error("missing --readonly flag")
	}
}

func TestMcpCmdFlags(t *testing.T) {
	cmd := mcpCmd()
	if cmd.Use != "mcp" {
		t.Errorf("Use = %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("missing Short description")
	}
}

// --- Board list with description ---

func TestBoardListCmd_WithDescription(t *testing.T) {
	dir := setupCLI(t)
	content := "---\nname: Test\ndescription: A test board\n---\n\n## Col\n"
	_ = os.WriteFile(filepath.Join(dir, "test.md"), []byte(content), 0644)
	suppressStdout(t)

	cmd := boardListCmd()
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

// --- Card show with metadata ---

func TestCardShowCmd_WithMetadata(t *testing.T) {
	dir := setupCLI(t)
	content := `---
name: Meta Board
---

## Backlog

- [ ] Rich card #frontend
  tags: backend, api
  assignee: alice
  priority: high
  due: 2026-03-25
  Body text here.
`
	_ = os.WriteFile(filepath.Join(dir, "metaboard.md"), []byte(content), 0644)
	suppressStdout(t)

	cmd := cardShowCmd()
	if err := cmd.RunE(cmd, []string{"metaboard", "0", "0"}); err != nil {
		t.Fatal(err)
	}
}

// --- Parent command construction tests ---

func TestBoardCmd_Subcommands(t *testing.T) {
	cmd := boardCmd()
	if cmd.Use != "board" {
		t.Errorf("Use = %q", cmd.Use)
	}
	names := map[string]bool{}
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}
	for _, want := range []string{"list", "create", "delete"} {
		if !names[want] {
			t.Errorf("missing subcommand %q", want)
		}
	}
}

func TestCardCmd_Subcommands(t *testing.T) {
	cmd := cardCmd()
	if cmd.Use != "card" {
		t.Errorf("Use = %q", cmd.Use)
	}
	names := map[string]bool{}
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}
	for _, want := range []string{"add", "move", "complete", "tag", "show", "delete"} {
		if !names[want] {
			t.Errorf("missing subcommand %q", want)
		}
	}
}

func TestColumnCmd_Subcommands(t *testing.T) {
	cmd := columnCmd()
	if cmd.Use != "column" {
		t.Errorf("Use = %q", cmd.Use)
	}
	names := map[string]bool{}
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}
	for _, want := range []string{"add", "move", "delete"} {
		if !names[want] {
			t.Errorf("missing subcommand %q", want)
		}
	}
}

// --- Card show completed card ---

func TestCardShowCmd_Completed(t *testing.T) {
	dir := setupCLI(t)
	content := "---\nname: Done Board\n---\n\n## Done\n\n- [x] Finished task\n"
	_ = os.WriteFile(filepath.Join(dir, "doneboard.md"), []byte(content), 0644)
	suppressStdout(t)

	cmd := cardShowCmd()
	if err := cmd.RunE(cmd, []string{"doneboard", "0", "0"}); err != nil {
		t.Fatal(err)
	}
}

// --- Serve command env var branches ---

func TestServeCmdFlags_WithEnvVars(t *testing.T) {
	t.Setenv("LIVEBOARD_HOST", "0.0.0.0")
	t.Setenv("LIVEBOARD_PORT", "9090")
	cmd := serveCmd()
	f := cmd.Flags().Lookup("host")
	if f == nil {
		t.Fatal("missing --host flag")
	}
	if f.DefValue != "0.0.0.0" {
		t.Errorf("host default = %q, want 0.0.0.0", f.DefValue)
	}
	pf := cmd.Flags().Lookup("port")
	if pf == nil {
		t.Fatal("missing --port flag")
	}
	if pf.DefValue != "9090" {
		t.Errorf("port default = %q, want 9090", pf.DefValue)
	}
}

func TestServeCmdFlags_WithInvalidPort(t *testing.T) {
	t.Setenv("LIVEBOARD_PORT", "notaport")
	cmd := serveCmd()
	pf := cmd.Flags().Lookup("port")
	if pf == nil {
		t.Fatal("missing --port flag")
	}
	// Invalid port should fallback to default 7070
	if pf.DefValue != "7070" {
		t.Errorf("port default = %q, want 7070", pf.DefValue)
	}
}

// --- PersistentPreRunE via rootCmd ---

func TestRootCmd_PersistentPreRunE(t *testing.T) {
	// Exercise main()'s PersistentPreRunE by building a root command
	// with --dir flag and running board list through it
	dir := t.TempDir()
	content := "---\nname: PreRun Test\n---\n\n## Col\n"
	_ = os.WriteFile(filepath.Join(dir, "prerun.md"), []byte(content), 0644)
	suppressStdout(t)

	// Reset global state
	workDir = ""
	ws = nil
	eng = nil

	rootCmd := &cobra.Command{
		Use:   "liveboard",
		Short: "Markdown-native, local-first Kanban system",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if workDir == "" {
				var cloud bool
				workDir, cloud = defaults.WorkDir()
				usingCloud = cloud
			}
			ws = workspace.Open(workDir)
			eng = board.New()
			return nil
		},
	}
	rootCmd.PersistentFlags().StringVarP(&workDir, "dir", "d", "", "workspace directory")
	rootCmd.AddCommand(boardCmd())

	rootCmd.SetArgs([]string{"--dir", dir, "board", "list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

// --- Error path tests for invalid args ---

func TestCardMoveCmd_InvalidCardIdx(t *testing.T) {
	suppressStdout(t)
	dir := setupCLI(t)
	createCLIBoard(t, dir, "myboard")

	cmd := cardMoveCmd()
	if err := cmd.RunE(cmd, []string{"myboard", "0", "notanum", "Done"}); err == nil {
		t.Error("expected error for non-numeric card_idx")
	}
}

func TestCardCompleteCmd_InvalidArgs(t *testing.T) {
	suppressStdout(t)
	dir := setupCLI(t)
	createCLIBoard(t, dir, "myboard")

	cmd := cardCompleteCmd()
	if err := cmd.RunE(cmd, []string{"myboard", "bad", "0"}); err == nil {
		t.Error("expected error for non-numeric col_idx")
	}

	cmd2 := cardCompleteCmd()
	if err := cmd2.RunE(cmd2, []string{"myboard", "0", "bad"}); err == nil {
		t.Error("expected error for non-numeric card_idx")
	}
}

func TestCardTagCmd_InvalidArgs(t *testing.T) {
	suppressStdout(t)
	dir := setupCLI(t)
	createCLIBoard(t, dir, "myboard")

	cmd := cardTagCmd()
	if err := cmd.RunE(cmd, []string{"myboard", "bad", "0", "tag"}); err == nil {
		t.Error("expected error for non-numeric col_idx")
	}

	cmd2 := cardTagCmd()
	if err := cmd2.RunE(cmd2, []string{"myboard", "0", "bad", "tag"}); err == nil {
		t.Error("expected error for non-numeric card_idx")
	}
}

func TestCardShowCmd_InvalidArgs(t *testing.T) {
	suppressStdout(t)
	dir := setupCLI(t)
	createCLIBoard(t, dir, "myboard")

	cmd := cardShowCmd()
	if err := cmd.RunE(cmd, []string{"myboard", "bad", "0"}); err == nil {
		t.Error("expected error for non-numeric col_idx")
	}

	cmd2 := cardShowCmd()
	if err := cmd2.RunE(cmd2, []string{"myboard", "0", "bad"}); err == nil {
		t.Error("expected error for non-numeric card_idx")
	}
}

func TestCardDeleteCmd_InvalidArgs(t *testing.T) {
	suppressStdout(t)
	dir := setupCLI(t)
	createCLIBoard(t, dir, "myboard")

	cmd := cardDeleteCmd()
	if err := cmd.RunE(cmd, []string{"myboard", "bad", "0"}); err == nil {
		t.Error("expected error for non-numeric col_idx")
	}

	cmd2 := cardDeleteCmd()
	if err := cmd2.RunE(cmd2, []string{"myboard", "0", "bad"}); err == nil {
		t.Error("expected error for non-numeric card_idx")
	}
}

func TestCardMoveCmd_InvalidColIdx(t *testing.T) {
	suppressStdout(t)
	dir := setupCLI(t)
	createCLIBoard(t, dir, "myboard")

	cmd := cardMoveCmd()
	if err := cmd.RunE(cmd, []string{"myboard", "notanum", "0", "Done"}); err == nil {
		t.Error("expected error for non-numeric col_idx")
	}
}
