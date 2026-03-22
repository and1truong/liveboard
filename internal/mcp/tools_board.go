package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type listBoardsInput struct{}

type getBoardInput struct {
	Board string `json:"board" jsonschema:"board name (filename without .md extension)"`
}

type createBoardInput struct {
	Name string `json:"name" jsonschema:"name for the new board"`
}

type deleteBoardInput struct {
	Board string `json:"board" jsonschema:"board name to delete"`
}

// boardSummary is a lightweight view returned by list_boards.
type boardSummary struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Icon        string   `json:"icon,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Columns     int      `json:"columns"`
	Cards       int      `json:"cards"`
}

// indexedBoard adds explicit indices so the LLM can address items.
type indexedBoard struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Icon        string          `json:"icon,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
	Members     []string        `json:"members,omitempty"`
	Version     int             `json:"version"`
	Columns     []indexedColumn `json:"columns"`
}

type indexedColumn struct {
	Index int           `json:"index"`
	Name  string        `json:"name"`
	Cards []indexedCard `json:"cards"`
}

type indexedCard struct {
	Index     int      `json:"index"`
	Title     string   `json:"title"`
	Completed bool     `json:"completed"`
	Tags      []string `json:"tags,omitempty"`
	Assignee  string   `json:"assignee,omitempty"`
	Priority  string   `json:"priority,omitempty"`
	Due       string   `json:"due,omitempty"`
	Body      string   `json:"body,omitempty"`
}

func (m *MCPServer) registerBoardTools() {
	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "list_boards",
		Description: "List all boards in the workspace with summary info",
		Annotations: &mcpsdk.ToolAnnotations{ReadOnlyHint: true},
	}, func(_ context.Context, _ *mcpsdk.CallToolRequest, _ listBoardsInput) (*mcpsdk.CallToolResult, any, error) {
		boards, err := m.ws.ListBoards()
		if err != nil {
			return errResult(err)
		}
		summaries := make([]boardSummary, len(boards))
		for i, b := range boards {
			total := 0
			for _, c := range b.Columns {
				total += len(c.Cards)
			}
			summaries[i] = boardSummary{
				Name:        b.Name,
				Description: b.Description,
				Icon:        b.Icon,
				Tags:        b.Tags,
				Columns:     len(b.Columns),
				Cards:       total,
			}
		}
		return jsonResult(summaries)
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "get_board",
		Description: "Get full board contents with indexed columns and cards. Use the indices for card/column operations.",
		Annotations: &mcpsdk.ToolAnnotations{ReadOnlyHint: true},
	}, func(_ context.Context, _ *mcpsdk.CallToolRequest, args getBoardInput) (*mcpsdk.CallToolResult, any, error) {
		b, err := m.ws.LoadBoard(args.Board)
		if err != nil {
			return errResult(err)
		}
		ib := indexedBoard{
			Name:        b.Name,
			Description: b.Description,
			Icon:        b.Icon,
			Tags:        b.Tags,
			Members:     b.Members,
			Version:     b.Version,
			Columns:     make([]indexedColumn, len(b.Columns)),
		}
		for ci, col := range b.Columns {
			ic := indexedColumn{Index: ci, Name: col.Name, Cards: make([]indexedCard, len(col.Cards))}
			for cdi, card := range col.Cards {
				ic.Cards[cdi] = indexedCard{
					Index:     cdi,
					Title:     card.Title,
					Completed: card.Completed,
					Tags:      card.Tags,
					Assignee:  card.Assignee,
					Priority:  card.Priority,
					Due:       card.Due,
					Body:      card.Body,
				}
			}
			ib.Columns[ci] = ic
		}
		return jsonResult(ib)
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "create_board",
		Description: "Create a new board in the workspace",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, func(_ context.Context, _ *mcpsdk.CallToolRequest, args createBoardInput) (*mcpsdk.CallToolResult, any, error) {
		b, err := m.ws.CreateBoard(args.Name)
		if err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("Created board %q", b.Name))
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "delete_board",
		Description: "Delete a board from the workspace (irreversible)",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(true)},
	}, func(_ context.Context, _ *mcpsdk.CallToolRequest, args deleteBoardInput) (*mcpsdk.CallToolResult, any, error) {
		if err := m.ws.DeleteBoard(args.Board); err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("Deleted board %q", args.Board))
	})
}
