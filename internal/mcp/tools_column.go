package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type addColumnInput struct {
	Board string `json:"board" jsonschema:"board name"`
	Name  string `json:"name" jsonschema:"new column name"`
}

type deleteColumnInput struct {
	Board string `json:"board" jsonschema:"board name"`
	Name  string `json:"name" jsonschema:"column name to delete"`
}

type renameColumnInput struct {
	Board   string `json:"board" jsonschema:"board name"`
	OldName string `json:"old_name" jsonschema:"current column name"`
	NewName string `json:"new_name" jsonschema:"new column name"`
}

type moveColumnInput struct {
	Board string `json:"board" jsonschema:"board name"`
	Name  string `json:"name" jsonschema:"column name to move"`
	After string `json:"after" jsonschema:"place after this column (empty for first position)"`
}

type sortColumnInput struct {
	Board       string `json:"board" jsonschema:"board name"`
	ColumnIndex int    `json:"column_index" jsonschema:"0-based column index"`
	SortBy      string `json:"sort_by" jsonschema:"sort key: name or priority or due"`
}

func (m *Server) registerColumnTools() {
	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "add_column",
		Description: "Add a new column to a board",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, func(_ context.Context, _ *mcpsdk.CallToolRequest, args addColumnInput) (*mcpsdk.CallToolResult, any, error) {
		path := m.ws.BoardPath(args.Board)
		if err := m.eng.AddColumn(path, args.Name); err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("Added column %q", args.Name))
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "delete_column",
		Description: "Delete a column and all its cards from a board (irreversible)",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(true)},
	}, func(_ context.Context, _ *mcpsdk.CallToolRequest, args deleteColumnInput) (*mcpsdk.CallToolResult, any, error) {
		path := m.ws.BoardPath(args.Board)
		if err := m.eng.DeleteColumn(path, args.Name); err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("Deleted column %q", args.Name))
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "rename_column",
		Description: "Rename a column on a board",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, func(_ context.Context, _ *mcpsdk.CallToolRequest, args renameColumnInput) (*mcpsdk.CallToolResult, any, error) {
		path := m.ws.BoardPath(args.Board)
		if err := m.eng.RenameColumn(path, args.OldName, args.NewName); err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("Renamed column %q to %q", args.OldName, args.NewName))
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "move_column",
		Description: "Move a column to a new position on a board",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, func(_ context.Context, _ *mcpsdk.CallToolRequest, args moveColumnInput) (*mcpsdk.CallToolResult, any, error) {
		path := m.ws.BoardPath(args.Board)
		if err := m.eng.MoveColumn(path, args.Name, args.After); err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("Moved column %q", args.Name))
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "sort_column",
		Description: "Sort cards within a column by name, priority, or due date",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, func(_ context.Context, _ *mcpsdk.CallToolRequest, args sortColumnInput) (*mcpsdk.CallToolResult, any, error) {
		path := m.ws.BoardPath(args.Board)
		if err := m.eng.SortColumn(path, args.ColumnIndex, args.SortBy); err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("Column sorted by %s", args.SortBy))
	})
}
