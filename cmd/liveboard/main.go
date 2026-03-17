package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/and1truong/liveboard/internal/board"
	gitpkg "github.com/and1truong/liveboard/internal/git"
	"github.com/and1truong/liveboard/internal/workspace"
	"github.com/and1truong/liveboard/pkg/models"
	"github.com/spf13/cobra"
)

var (
	workDir string
	ws      *workspace.Workspace
	eng     *board.Engine
	gitRepo *gitpkg.Repository
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "liveboard",
		Short: "Markdown-native, local-first Kanban system",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if workDir == "" {
				workDir, _ = os.Getwd()
			}
			ws = workspace.Open(workDir)
			eng = board.New()

			var err error
			gitRepo, err = gitpkg.Open(workDir, true)
			if err != nil {
				// Non-fatal: git features just won't work.
				gitRepo = nil
			}
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVarP(&workDir, "dir", "d", "", "workspace directory (default: current directory)")

	rootCmd.AddCommand(boardCmd())
	rootCmd.AddCommand(cardCmd())
	rootCmd.AddCommand(columnCmd())
	rootCmd.AddCommand(serveCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// --- Board commands ---

func boardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "board",
		Short: "Manage boards",
	}
	cmd.AddCommand(boardListCmd())
	cmd.AddCommand(boardCreateCmd())
	cmd.AddCommand(boardDeleteCmd())
	return cmd
}

func boardListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all boards",
		RunE: func(cmd *cobra.Command, args []string) error {
			boards, err := ws.ListBoards()
			if err != nil {
				return err
			}
			if len(boards) == 0 {
				fmt.Println("No boards found.")
				return nil
			}
			for _, b := range boards {
				cardCount := 0
				for _, col := range b.Columns {
					cardCount += len(col.Cards)
				}
				desc := ""
				if b.Description != "" {
					desc = " — " + b.Description
				}
				fmt.Printf("  %s%s (%d cards, %d columns)\n", b.Name, desc, cardCount, len(b.Columns))
			}
			return nil
		},
	}
}

func boardCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new board",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			b, err := ws.CreateBoard(name)
			if err != nil {
				return err
			}
			fmt.Printf("Created board %q with columns: %s\n", name, columnNames(b))
			gitCommit(name+".md", fmt.Sprintf("board: create %q", name))
			return nil
		},
	}
}

func boardDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a board",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			relPath := name + ".md"
			if err := ws.DeleteBoard(name); err != nil {
				return err
			}
			fmt.Printf("Deleted board %q\n", name)
			gitCommitRemove(relPath, fmt.Sprintf("board: delete %q", name))
			return nil
		},
	}
}

// --- Card commands ---

func cardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "card",
		Short: "Manage cards",
	}
	cmd.AddCommand(cardAddCmd())
	cmd.AddCommand(cardMoveCmd())
	cmd.AddCommand(cardCompleteCmd())
	cmd.AddCommand(cardTagCmd())
	cmd.AddCommand(cardShowCmd())
	cmd.AddCommand(cardDeleteCmd())
	return cmd
}

func cardAddCmd() *cobra.Command {
	var column string
	cmd := &cobra.Command{
		Use:   "add <board> <title>",
		Short: "Add a card to a board",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			boardName, title := args[0], args[1]
			path := ws.BoardPath(boardName)

			if column == "" {
				column = "Backlog"
			}

			card, err := eng.AddCard(path, column, title)
			if err != nil {
				return err
			}
			fmt.Printf("Added card %q → %s [%s]\n", title, column, card.ID)
			gitCommit(boardName+".md", fmt.Sprintf("card: add %q → %s", title, column))
			return nil
		},
	}
	cmd.Flags().StringVarP(&column, "column", "c", "", "target column (default: Backlog)")
	return cmd
}

func cardMoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "move <id> <column>",
		Short: "Move a card to another column",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cardID, targetCol := args[0], args[1]
			b, err := ws.FindBoardByCardID(cardID)
			if err != nil {
				return err
			}
			if err := eng.MoveCard(b.FilePath, cardID, targetCol); err != nil {
				return err
			}
			fmt.Printf("Moved card %s → %s\n", shortID(cardID), targetCol)
			relPath := filepath.Base(b.FilePath)
			gitCommit(relPath, fmt.Sprintf("card: move %s → %s", shortID(cardID), targetCol))
			return nil
		},
	}
}

func cardCompleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "complete <id>",
		Short: "Mark a card as completed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cardID := args[0]
			b, err := ws.FindBoardByCardID(cardID)
			if err != nil {
				return err
			}
			if err := eng.CompleteCard(b.FilePath, cardID); err != nil {
				return err
			}
			fmt.Printf("Completed card %s\n", shortID(cardID))
			relPath := filepath.Base(b.FilePath)
			gitCommit(relPath, fmt.Sprintf("card: complete %s", shortID(cardID)))
			return nil
		},
	}
}

func cardTagCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tag <id> <tag> [tag...]",
		Short: "Add tags to a card",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cardID := args[0]
			tags := args[1:]
			b, err := ws.FindBoardByCardID(cardID)
			if err != nil {
				return err
			}
			if err := eng.TagCard(b.FilePath, cardID, tags); err != nil {
				return err
			}
			fmt.Printf("Tagged card %s with: %s\n", shortID(cardID), strings.Join(tags, ", "))
			relPath := filepath.Base(b.FilePath)
			gitCommit(relPath, fmt.Sprintf("card: tag %s [%s]", shortID(cardID), strings.Join(tags, ", ")))
			return nil
		},
	}
}

func cardShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show card details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cardID := args[0]
			b, err := ws.FindBoardByCardID(cardID)
			if err != nil {
				return err
			}
			card, colName, err := eng.ShowCard(b.FilePath, cardID)
			if err != nil {
				return err
			}
			status := "[ ]"
			if card.Completed {
				status = "[x]"
			}
			fmt.Printf("%s %s\n", status, card.Title)
			fmt.Printf("  ID:     %s\n", card.ID)
			fmt.Printf("  Board:  %s\n", b.Name)
			fmt.Printf("  Column: %s\n", colName)
			if len(card.Tags) > 0 {
				fmt.Printf("  Tags:   %s\n", strings.Join(card.Tags, ", "))
			}
			if card.Assignee != "" {
				fmt.Printf("  Assign: %s\n", card.Assignee)
			}
			if card.Priority != "" {
				fmt.Printf("  Priority: %s\n", card.Priority)
			}
			if card.Due != "" {
				fmt.Printf("  Due:    %s\n", card.Due)
			}
			return nil
		},
	}
}

func cardDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a card",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cardID := args[0]
			b, err := ws.FindBoardByCardID(cardID)
			if err != nil {
				return err
			}
			if err := eng.DeleteCard(b.FilePath, cardID); err != nil {
				return err
			}
			fmt.Printf("Deleted card %s\n", shortID(cardID))
			relPath := filepath.Base(b.FilePath)
			gitCommit(relPath, fmt.Sprintf("card: delete %s", shortID(cardID)))
			return nil
		},
	}
}

// --- Column commands ---

func columnCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "column",
		Short: "Manage columns",
	}
	cmd.AddCommand(columnAddCmd())
	cmd.AddCommand(columnMoveCmd())
	cmd.AddCommand(columnDeleteCmd())
	return cmd
}

func columnAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <board> <name>",
		Short: "Add a column to a board",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			boardName, colName := args[0], args[1]
			path := ws.BoardPath(boardName)
			if err := eng.AddColumn(path, colName); err != nil {
				return err
			}
			fmt.Printf("Added column %q to %s\n", colName, boardName)
			gitCommit(boardName+".md", fmt.Sprintf("column: add %q to %s", colName, boardName))
			return nil
		},
	}
}

func columnMoveCmd() *cobra.Command {
	var after string
	cmd := &cobra.Command{
		Use:   "move <board> <name>",
		Short: "Move a column",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			boardName, colName := args[0], args[1]
			path := ws.BoardPath(boardName)
			if err := eng.MoveColumn(path, colName, after); err != nil {
				return err
			}
			fmt.Printf("Moved column %q after %q in %s\n", colName, after, boardName)
			gitCommit(boardName+".md", fmt.Sprintf("column: move %q after %q in %s", colName, after, boardName))
			return nil
		},
	}
	cmd.Flags().StringVar(&after, "after", "", "place column after this column")
	_ = cmd.MarkFlagRequired("after")
	return cmd
}

func columnDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <board> <name>",
		Short: "Delete a column from a board",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			boardName, colName := args[0], args[1]
			path := ws.BoardPath(boardName)
			if err := eng.DeleteColumn(path, colName); err != nil {
				return err
			}
			fmt.Printf("Deleted column %q from %s\n", colName, boardName)
			gitCommit(boardName+".md", fmt.Sprintf("column: delete %q from %s", colName, boardName))
			return nil
		},
	}
}

// --- Helpers ---

func gitCommit(relPath, message string) {
	if gitRepo != nil {
		_ = gitRepo.Commit(relPath, message)
	}
}

func gitCommitRemove(relPath, message string) {
	if gitRepo != nil {
		_ = gitRepo.CommitRemove(relPath, message)
	}
}

func columnNames(b *models.Board) string {
	var names []string
	for _, c := range b.Columns {
		names = append(names, c.Name)
	}
	return strings.Join(names, ", ")
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
