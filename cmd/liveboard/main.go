// Package main is the entry point for the liveboard CLI.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/internal/defaults"
	"github.com/and1truong/liveboard/internal/workspace"
	"github.com/and1truong/liveboard/pkg/models"
)

var (
	version = "dev"
	commit  = "none"
)

var (
	workDir string
	ws      *workspace.Workspace
	eng     *board.Engine
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "liveboard",
		Short: "Markdown-native, local-first Kanban system",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if workDir == "" {
				cfg := defaults.LoadCLIConfig()
				if cfg.Workspace != "" {
					workDir = cfg.Workspace
				} else {
					workDir, _ = defaults.WorkDir()
				}
			}
			ws = workspace.Open(workDir)
			eng = board.New()
			return nil
		},
	}

	rootCmd.Version = version + " (" + commit + ")"
	rootCmd.PersistentFlags().StringVarP(&workDir, "dir", "d", "", "workspace directory (default: current directory)")

	rootCmd.AddCommand(boardCmd())
	rootCmd.AddCommand(cardCmd())
	rootCmd.AddCommand(columnCmd())
	rootCmd.AddCommand(serveCmd())
	rootCmd.AddCommand(mcpCmd())
	rootCmd.AddCommand(exportCmd())
	rootCmd.AddCommand(gcCmd())

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
		RunE: func(_ *cobra.Command, _ []string) error {
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
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			b, err := ws.CreateBoard(name)
			if err != nil {
				return err
			}
			fmt.Printf("Created board %q with columns: %s\n", name, columnNames(b))
			return nil
		},
	}
}

func boardDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a board",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			if err := ws.DeleteBoard(name); err != nil {
				return err
			}
			fmt.Printf("Deleted board %q\n", name)
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
		RunE: func(_ *cobra.Command, args []string) error {
			boardName, title := args[0], args[1]
			path, err := ws.BoardPath(boardName)
			if err != nil {
				return err
			}

			if column == "" {
				column = "Backlog"
			}

			card, err := eng.AddCard(path, column, title, false)
			if err != nil {
				return err
			}
			fmt.Printf("Added card %q → %s\n", card.Title, column)
			return nil
		},
	}
	cmd.Flags().StringVarP(&column, "column", "c", "", "target column (default: Backlog)")
	return cmd
}

func cardMoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "move <board> <col_idx> <card_idx> <column>",
		Short: "Move a card to another column",
		Args:  cobra.ExactArgs(4),
		RunE: func(_ *cobra.Command, args []string) error {
			boardName := args[0]
			colIdx, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid col_idx: %w", err)
			}
			cardIdx, err := strconv.Atoi(args[2])
			if err != nil {
				return fmt.Errorf("invalid card_idx: %w", err)
			}
			targetCol := args[3]
			path, err := ws.BoardPath(boardName)
			if err != nil {
				return err
			}
			if err := eng.MoveCard(path, colIdx, cardIdx, targetCol); err != nil {
				return err
			}
			fmt.Printf("Moved card → %s\n", targetCol)
			return nil
		},
	}
}

func cardCompleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "complete <board> <col_idx> <card_idx>",
		Short: "Mark a card as completed",
		Args:  cobra.ExactArgs(3),
		RunE: func(_ *cobra.Command, args []string) error {
			boardName := args[0]
			colIdx, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid col_idx: %w", err)
			}
			cardIdx, err := strconv.Atoi(args[2])
			if err != nil {
				return fmt.Errorf("invalid card_idx: %w", err)
			}
			path, err := ws.BoardPath(boardName)
			if err != nil {
				return err
			}
			if err := eng.CompleteCard(path, colIdx, cardIdx); err != nil {
				return err
			}
			fmt.Printf("Toggled card completion\n")
			return nil
		},
	}
}

func cardTagCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tag <board> <col_idx> <card_idx> <tag> [tag...]",
		Short: "Add tags to a card",
		Args:  cobra.MinimumNArgs(4),
		RunE: func(_ *cobra.Command, args []string) error {
			boardName := args[0]
			colIdx, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid col_idx: %w", err)
			}
			cardIdx, err := strconv.Atoi(args[2])
			if err != nil {
				return fmt.Errorf("invalid card_idx: %w", err)
			}
			tags := args[3:]
			path, err := ws.BoardPath(boardName)
			if err != nil {
				return err
			}
			if err := eng.TagCard(path, colIdx, cardIdx, tags); err != nil {
				return err
			}
			fmt.Printf("Tagged card with: %s\n", strings.Join(tags, ", "))
			return nil
		},
	}
}

func cardShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <board> <col_idx> <card_idx>",
		Short: "Show card details",
		Args:  cobra.ExactArgs(3),
		RunE: func(_ *cobra.Command, args []string) error {
			boardName := args[0]
			colIdx, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid col_idx: %w", err)
			}
			cardIdx, err := strconv.Atoi(args[2])
			if err != nil {
				return fmt.Errorf("invalid card_idx: %w", err)
			}
			path, err := ws.BoardPath(boardName)
			if err != nil {
				return err
			}
			card, colName, err := eng.ShowCard(path, colIdx, cardIdx)
			if err != nil {
				return err
			}
			status := "[ ]"
			if card.Completed {
				status = "[x]"
			}
			fmt.Printf("%s %s\n", status, card.Title)
			fmt.Printf("  Board:  %s\n", boardName)
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
		Use:   "delete <board> <col_idx> <card_idx>",
		Short: "Delete a card",
		Args:  cobra.ExactArgs(3),
		RunE: func(_ *cobra.Command, args []string) error {
			boardName := args[0]
			colIdx, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid col_idx: %w", err)
			}
			cardIdx, err := strconv.Atoi(args[2])
			if err != nil {
				return fmt.Errorf("invalid card_idx: %w", err)
			}
			path, err := ws.BoardPath(boardName)
			if err != nil {
				return err
			}
			if err := eng.DeleteCard(path, colIdx, cardIdx); err != nil {
				return err
			}
			fmt.Printf("Deleted card\n")
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
		RunE: func(_ *cobra.Command, args []string) error {
			boardName, colName := args[0], args[1]
			path, err := ws.BoardPath(boardName)
			if err != nil {
				return err
			}
			if err := eng.AddColumn(path, colName); err != nil {
				return err
			}
			fmt.Printf("Added column %q to %s\n", colName, boardName)
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
		RunE: func(_ *cobra.Command, args []string) error {
			boardName, colName := args[0], args[1]
			path, err := ws.BoardPath(boardName)
			if err != nil {
				return err
			}
			if err := eng.MoveColumn(path, colName, after); err != nil {
				return err
			}
			fmt.Printf("Moved column %q after %q in %s\n", colName, after, boardName)
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
		RunE: func(_ *cobra.Command, args []string) error {
			boardName, colName := args[0], args[1]
			path, err := ws.BoardPath(boardName)
			if err != nil {
				return err
			}
			if err := eng.DeleteColumn(path, colName); err != nil {
				return err
			}
			fmt.Printf("Deleted column %q from %s\n", colName, boardName)
			return nil
		},
	}
}

// --- Helpers ---

func columnNames(b *models.Board) string {
	var names []string
	for _, c := range b.Columns {
		names = append(names, c.Name)
	}
	return strings.Join(names, ", ")
}
