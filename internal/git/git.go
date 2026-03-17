package git

import (
	"fmt"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Repository wraps go-git for auto-commit operations.
type Repository struct {
	repo       *gogit.Repository
	autoCommit bool
}

// Open opens an existing Git repository at the given path.
// If the path is not a Git repo, it initializes one.
func Open(path string, autoCommit bool) (*Repository, error) {
	repo, err := gogit.PlainOpen(path)
	if err != nil {
		repo, err = gogit.PlainInit(path, false)
		if err != nil {
			return nil, fmt.Errorf("init git repo: %w", err)
		}
	}
	return &Repository{repo: repo, autoCommit: autoCommit}, nil
}

// Commit stages the given file and creates a commit with the given message.
// If autoCommit is disabled, this is a no-op.
func (r *Repository) Commit(filePath string, message string) error {
	if !r.autoCommit {
		return nil
	}

	wt, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}

	if _, err := wt.Add(filePath); err != nil {
		return fmt.Errorf("git add %s: %w", filePath, err)
	}

	_, err = wt.Commit(message, &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "liveboard",
			Email: "liveboard@local",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	return nil
}

// CommitRemove stages a file removal and commits.
func (r *Repository) CommitRemove(filePath string, message string) error {
	if !r.autoCommit {
		return nil
	}

	wt, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}

	if _, err := wt.Remove(filePath); err != nil {
		return fmt.Errorf("git rm %s: %w", filePath, err)
	}

	_, err = wt.Commit(message, &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "liveboard",
			Email: "liveboard@local",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	return nil
}
