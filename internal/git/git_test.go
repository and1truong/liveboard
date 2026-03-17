package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpen_InitializesNewRepo(t *testing.T) {
	dir := t.TempDir()
	repo, err := Open(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if repo == nil {
		t.Fatal("expected non-nil repository")
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		if os.IsNotExist(err) {
			t.Error(".git directory not created")
		} else {
			t.Fatalf("stat .git failed: %v", err)
		}
	}
}

func TestOpen_OpensExistingRepo(t *testing.T) {
	dir := t.TempDir()
	// First call initializes the repo.
	if _, err := Open(dir, false); err != nil {
		t.Fatal(err)
	}
	// Second call should open the existing repo without error.
	repo, err := Open(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if repo == nil {
		t.Fatal("expected non-nil repository")
	}
}

func TestCommit_NoOpWhenAutoCommitDisabled(t *testing.T) {
	dir := t.TempDir()
	repo, err := Open(dir, false)
	if err != nil {
		t.Fatal(err)
	}

	filePath := "test.md"
	if err := os.WriteFile(filepath.Join(dir, filePath), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := repo.Commit(filePath, "test commit"); err != nil {
		t.Fatal(err)
	}

	// No commit should have been created.
	if _, err := repo.repo.Head(); err == nil {
		t.Error("expected no HEAD commit when autoCommit is disabled")
	}
}

func TestCommit_CreatesCommit(t *testing.T) {
	dir := t.TempDir()
	repo, err := Open(dir, true)
	if err != nil {
		t.Fatal(err)
	}

	filePath := "board.md"
	if err := os.WriteFile(filepath.Join(dir, filePath), []byte("# Board\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := repo.Commit(filePath, "add board"); err != nil {
		t.Fatal(err)
	}

	ref, err := repo.repo.Head()
	if err != nil {
		t.Fatal(err)
	}
	commit, err := repo.repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatal(err)
	}
	if commit.Message != "add board" {
		t.Errorf("message = %q, want %q", commit.Message, "add board")
	}
	if commit.Author.Name != "liveboard" {
		t.Errorf("author name = %q, want %q", commit.Author.Name, "liveboard")
	}
	if commit.Author.Email != "liveboard@local" {
		t.Errorf("author email = %q, want %q", commit.Author.Email, "liveboard@local")
	}
}

func TestCommit_MultipleCommits(t *testing.T) {
	dir := t.TempDir()
	repo, err := Open(dir, true)
	if err != nil {
		t.Fatal(err)
	}

	for i, name := range []string{"a.md", "b.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
		msg := filepath.Base(name)
		if err := repo.Commit(name, msg); err != nil {
			t.Fatalf("commit %d: %v", i, err)
		}
	}

	ref, err := repo.repo.Head()
	if err != nil {
		t.Fatal(err)
	}
	commit, err := repo.repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatal(err)
	}
	if commit.Message != "b.md" {
		t.Errorf("latest commit message = %q, want %q", commit.Message, "b.md")
	}
}

func TestCommitRemove_NoOpWhenAutoCommitDisabled(t *testing.T) {
	dir := t.TempDir()
	repo, err := Open(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	// Should be a no-op; no error expected.
	if err := repo.CommitRemove("nonexistent.md", "remove"); err != nil {
		t.Fatal(err)
	}
}

func TestCommitRemove_FileRemainsWhenAutoCommitDisabled(t *testing.T) {
	dir := t.TempDir()
	repo, err := Open(dir, false)
	if err != nil {
		t.Fatal(err)
	}

	filePath := "board.md"
	fullPath := filepath.Join(dir, filePath)
	if err := os.WriteFile(fullPath, []byte("# Board\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// CommitRemove should be a no-op when autoCommit is disabled.
	if err := repo.CommitRemove(filePath, "remove"); err != nil {
		t.Fatal(err)
	}

	// File must still exist on disk.
	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			t.Error("file was removed despite autoCommit being disabled")
		} else {
			t.Fatalf("stat file failed: %v", err)
		}
	}
}

func TestCommitRemove_RemovesAndCommits(t *testing.T) {
	dir := t.TempDir()
	repo, err := Open(dir, true)
	if err != nil {
		t.Fatal(err)
	}

	filePath := "board.md"
	fullPath := filepath.Join(dir, filePath)

	// Create and commit the file.
	if err := os.WriteFile(fullPath, []byte("# Board\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := repo.Commit(filePath, "add board"); err != nil {
		t.Fatal(err)
	}

	// Remove and commit.
	if err := repo.CommitRemove(filePath, "remove board"); err != nil {
		t.Fatal(err)
	}

	// File should be gone from disk.
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Error("expected file to be removed from disk")
	}

	// Latest commit should reflect removal.
	ref, err := repo.repo.Head()
	if err != nil {
		t.Fatal(err)
	}
	commit, err := repo.repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatal(err)
	}
	if commit.Message != "remove board" {
		t.Errorf("message = %q, want %q", commit.Message, "remove board")
	}
}
