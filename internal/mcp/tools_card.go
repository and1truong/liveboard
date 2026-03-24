package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type addCardInput struct {
	Board   string `json:"board" jsonschema:"board name"`
	Column  string `json:"column" jsonschema:"column name to add card to"`
	Title   string `json:"title" jsonschema:"card title"`
	Prepend bool   `json:"prepend,omitempty" jsonschema:"add to top of column instead of bottom"`
}

type showCardInput struct {
	Board       string `json:"board" jsonschema:"board name"`
	ColumnIndex int    `json:"column_index" jsonschema:"0-based column index"`
	CardIndex   int    `json:"card_index" jsonschema:"0-based card index within column"`
}

type editCardInput struct {
	Board       string   `json:"board" jsonschema:"board name"`
	ColumnIndex int      `json:"column_index" jsonschema:"0-based column index"`
	CardIndex   int      `json:"card_index" jsonschema:"0-based card index within column"`
	Title       string   `json:"title,omitempty" jsonschema:"new title (empty keeps current)"`
	Body        string   `json:"body,omitempty" jsonschema:"card body text"`
	Tags        []string `json:"tags,omitempty" jsonschema:"tags (replaces all existing)"`
	Priority    string   `json:"priority,omitempty" jsonschema:"priority: critical high medium low"`
	Due         string   `json:"due,omitempty" jsonschema:"due date YYYY-MM-DD"`
	Assignee    string   `json:"assignee,omitempty" jsonschema:"assignee name"`
}

type moveCardInput struct {
	Board        string `json:"board" jsonschema:"board name"`
	ColumnIndex  int    `json:"column_index" jsonschema:"0-based source column index"`
	CardIndex    int    `json:"card_index" jsonschema:"0-based card index within source column"`
	TargetColumn string `json:"target_column" jsonschema:"destination column name"`
}

type completeCardInput struct {
	Board       string `json:"board" jsonschema:"board name"`
	ColumnIndex int    `json:"column_index" jsonschema:"0-based column index"`
	CardIndex   int    `json:"card_index" jsonschema:"0-based card index within column"`
}

type deleteCardInput struct {
	Board       string `json:"board" jsonschema:"board name"`
	ColumnIndex int    `json:"column_index" jsonschema:"0-based column index"`
	CardIndex   int    `json:"card_index" jsonschema:"0-based card index within column"`
}

func (m *Server) registerCardTools() {
	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "add_card",
		Description: "Add a new card to a column on a board",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, func(_ context.Context, _ *mcpsdk.CallToolRequest, args addCardInput) (*mcpsdk.CallToolResult, any, error) {
		path := m.ws.BoardPath(args.Board)
		card, err := m.eng.AddCard(path, args.Column, args.Title, args.Prepend)
		if err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("Added card %q to column %q", card.Title, args.Column))
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "show_card",
		Description: "Show full details of a card including body and metadata",
		Annotations: &mcpsdk.ToolAnnotations{ReadOnlyHint: true},
	}, func(_ context.Context, _ *mcpsdk.CallToolRequest, args showCardInput) (*mcpsdk.CallToolResult, any, error) {
		path := m.ws.BoardPath(args.Board)
		card, colName, err := m.eng.ShowCard(path, args.ColumnIndex, args.CardIndex)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]any{
			"column": colName,
			"card":   card,
		})
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "edit_card",
		Description: "Edit a card's title, body, tags, priority, due date, or assignee. Only provided fields are updated.",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, func(_ context.Context, _ *mcpsdk.CallToolRequest, args editCardInput) (*mcpsdk.CallToolResult, any, error) {
		path := m.ws.BoardPath(args.Board)
		err := m.eng.EditCard(path, args.ColumnIndex, args.CardIndex,
			args.Title, args.Body, args.Tags, args.Priority, args.Due, args.Assignee)
		if err != nil {
			return errResult(err)
		}
		return textResult("Card updated")
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "move_card",
		Description: "Move a card to a different column",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, func(_ context.Context, _ *mcpsdk.CallToolRequest, args moveCardInput) (*mcpsdk.CallToolResult, any, error) {
		path := m.ws.BoardPath(args.Board)
		if err := m.eng.MoveCard(path, args.ColumnIndex, args.CardIndex, args.TargetColumn); err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("Card moved to column %q", args.TargetColumn))
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "complete_card",
		Description: "Toggle a card's completion status",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, func(_ context.Context, _ *mcpsdk.CallToolRequest, args completeCardInput) (*mcpsdk.CallToolResult, any, error) {
		path := m.ws.BoardPath(args.Board)
		if err := m.eng.CompleteCard(path, args.ColumnIndex, args.CardIndex); err != nil {
			return errResult(err)
		}
		return textResult("Card completion toggled")
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "delete_card",
		Description: "Delete a card from a board (irreversible)",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(true)},
	}, func(_ context.Context, _ *mcpsdk.CallToolRequest, args deleteCardInput) (*mcpsdk.CallToolResult, any, error) {
		path := m.ws.BoardPath(args.Board)
		if err := m.eng.DeleteCard(path, args.ColumnIndex, args.CardIndex); err != nil {
			return errResult(err)
		}
		return textResult("Card deleted")
	})
}
