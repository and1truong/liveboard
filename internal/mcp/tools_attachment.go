package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/pkg/models"
)

type addAttachmentRefInput struct {
	Board       string `json:"board" jsonschema:"board name"`
	ColumnIndex int    `json:"column_index" jsonschema:"0-based column index"`
	CardIndex   int    `json:"card_index" jsonschema:"0-based card index within column"`
	Hash        string `json:"hash" jsonschema:"<sha256-hex>.<ext> — must already exist in the workspace pool (uploaded via HTTP /api/v1/attachments)"`
	Name        string `json:"name" jsonschema:"display filename shown in UI and downloads"`
	Size        int64  `json:"size" jsonschema:"file size in bytes"`
	Mime        string `json:"mime" jsonschema:"sniffed MIME type"`
}

type removeAttachmentInput struct {
	Board       string `json:"board" jsonschema:"board name"`
	ColumnIndex int    `json:"column_index" jsonschema:"0-based column index"`
	CardIndex   int    `json:"card_index" jsonschema:"0-based card index within column"`
	Hash        string `json:"hash" jsonschema:"hash of attachment to remove"`
}

type moveAttachmentInput struct {
	Board    string `json:"board" jsonschema:"board name"`
	FromCol  int    `json:"from_col" jsonschema:"source column index"`
	FromCard int    `json:"from_card" jsonschema:"source card index"`
	ToCol    int    `json:"to_col" jsonschema:"destination column index (same board)"`
	ToCard   int    `json:"to_card" jsonschema:"destination card index"`
	Hash     string `json:"hash" jsonschema:"hash of attachment to move"`
}

type renameAttachmentInput struct {
	Board       string `json:"board" jsonschema:"board name"`
	ColumnIndex int    `json:"column_index" jsonschema:"0-based column index"`
	CardIndex   int    `json:"card_index" jsonschema:"0-based card index within column"`
	Hash        string `json:"hash" jsonschema:"hash of attachment to rename"`
	NewName     string `json:"new_name" jsonschema:"new display filename"`
}

type reorderAttachmentsInput struct {
	Board         string   `json:"board" jsonschema:"board name"`
	ColumnIndex   int      `json:"column_index" jsonschema:"0-based column index"`
	CardIndex     int      `json:"card_index" jsonschema:"0-based card index within column"`
	HashesInOrder []string `json:"hashes_in_order" jsonschema:"hashes in their new order; unknown hashes are ignored; survivors not listed are appended in original order"`
}

func (m *Server) registerAttachmentTools() {
	m.registerAttachmentMutationTools()
	m.registerAttachmentOrganizeTools()
}

func (m *Server) registerAttachmentMutationTools() {
	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "card_add_attachment_ref",
		Description: "Add an existing-blob reference (descriptor only) to a card. Bytes must already be uploaded via the HTTP /api/v1/attachments endpoint; this tool only writes the descriptor.",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, func(_ context.Context, _ *mcpsdk.CallToolRequest, args addAttachmentRefInput) (*mcpsdk.CallToolResult, any, error) {
		path, err := m.ws.BoardPath(args.Board)
		if err != nil {
			return errResult(err)
		}
		op := board.MutationOp{
			Type: "add_attachments",
			AddAttachments: &board.AddAttachmentsOp{
				ColIdx:  args.ColumnIndex,
				CardIdx: args.CardIndex,
				Items: []models.Attachment{{
					Hash: args.Hash, Name: args.Name, Size: args.Size, Mime: args.Mime,
				}},
			},
		}
		if err := m.eng.MutateBoard(path, -1, func(b *models.Board) error {
			return board.ApplyMutation(b, op)
		}); err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("Added attachment %q (hash %s)", args.Name, args.Hash))
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "card_remove_attachment",
		Description: "Remove an attachment from a card by hash. Idempotent; missing hash is a no-op.",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(true)},
	}, func(_ context.Context, _ *mcpsdk.CallToolRequest, args removeAttachmentInput) (*mcpsdk.CallToolResult, any, error) {
		path, err := m.ws.BoardPath(args.Board)
		if err != nil {
			return errResult(err)
		}
		op := board.MutationOp{
			Type:             "remove_attachment",
			RemoveAttachment: &board.RemoveAttachmentOp{ColIdx: args.ColumnIndex, CardIdx: args.CardIndex, Hash: args.Hash},
		}
		if err := m.eng.MutateBoard(path, -1, func(b *models.Board) error {
			return board.ApplyMutation(b, op)
		}); err != nil {
			return errResult(err)
		}
		return textResult("Attachment removed")
	})
}

func (m *Server) registerAttachmentOrganizeTools() {
	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "card_move_attachment",
		Description: "Move an attachment between two cards on the same board. Cross-board moves should be done as add+remove client-side.",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, func(_ context.Context, _ *mcpsdk.CallToolRequest, args moveAttachmentInput) (*mcpsdk.CallToolResult, any, error) {
		path, err := m.ws.BoardPath(args.Board)
		if err != nil {
			return errResult(err)
		}
		op := board.MutationOp{
			Type: "move_attachment",
			MoveAttachment: &board.MoveAttachmentOp{
				FromCol: args.FromCol, FromCard: args.FromCard,
				ToCol: args.ToCol, ToCard: args.ToCard,
				Hash: args.Hash,
			},
		}
		if err := m.eng.MutateBoard(path, -1, func(b *models.Board) error {
			return board.ApplyMutation(b, op)
		}); err != nil {
			return errResult(err)
		}
		return textResult("Attachment moved")
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "card_rename_attachment",
		Description: "Update the display name of an attachment. The hash and on-disk blob are unchanged.",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, func(_ context.Context, _ *mcpsdk.CallToolRequest, args renameAttachmentInput) (*mcpsdk.CallToolResult, any, error) {
		path, err := m.ws.BoardPath(args.Board)
		if err != nil {
			return errResult(err)
		}
		op := board.MutationOp{
			Type:             "rename_attachment",
			RenameAttachment: &board.RenameAttachmentOp{ColIdx: args.ColumnIndex, CardIdx: args.CardIndex, Hash: args.Hash, NewName: args.NewName},
		}
		if err := m.eng.MutateBoard(path, -1, func(b *models.Board) error {
			return board.ApplyMutation(b, op)
		}); err != nil {
			return errResult(err)
		}
		return textResult("Attachment renamed")
	})

	mcpsdk.AddTool(m.server, &mcpsdk.Tool{
		Name:        "card_reorder_attachments",
		Description: "Reorder a card's attachments to match the given hash list. Unknown hashes are ignored; survivors not listed are appended in original order.",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, func(_ context.Context, _ *mcpsdk.CallToolRequest, args reorderAttachmentsInput) (*mcpsdk.CallToolResult, any, error) {
		path, err := m.ws.BoardPath(args.Board)
		if err != nil {
			return errResult(err)
		}
		op := board.MutationOp{
			Type:               "reorder_attachments",
			ReorderAttachments: &board.ReorderAttachmentsOp{ColIdx: args.ColumnIndex, CardIdx: args.CardIndex, HashesInOrder: args.HashesInOrder},
		}
		if err := m.eng.MutateBoard(path, -1, func(b *models.Board) error {
			return board.ApplyMutation(b, op)
		}); err != nil {
			return errResult(err)
		}
		return textResult("Attachments reordered")
	})
}
