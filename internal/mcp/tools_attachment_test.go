package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/and1truong/liveboard/internal/parser"
	"github.com/and1truong/liveboard/pkg/models"
)

func parseBoard(t *testing.T, dir string) *models.Board {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "test-board.md"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := parser.Parse(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestMCPAddAttachmentRef(t *testing.T) {
	srv, dir := setup(t)
	result := callTool(t, srv, "card_add_attachment_ref", map[string]any{
		"board":        "test-board",
		"column_index": 0,
		"card_index":   0,
		"hash":         "abc.pdf",
		"name":         "doc.pdf",
		"size":         int64(1),
		"mime":         "application/pdf",
	})
	text := resultText(t, result)
	if !strings.Contains(text, "Added attachment") {
		t.Errorf("unexpected response: %q", text)
	}

	b := parseBoard(t, dir)
	got := b.Columns[0].Cards[0].Attachments
	if len(got) != 1 || got[0].Hash != "abc.pdf" {
		t.Errorf("attachment not on card: %+v", got)
	}
}

func TestMCPRemoveAttachment(t *testing.T) {
	srv, dir := setup(t)
	// Add first.
	_ = callTool(t, srv, "card_add_attachment_ref", map[string]any{
		"board": "test-board", "column_index": 0, "card_index": 0,
		"hash": "abc.pdf", "name": "doc.pdf", "size": int64(1), "mime": "application/pdf",
	})
	// Then remove.
	_ = callTool(t, srv, "card_remove_attachment", map[string]any{
		"board": "test-board", "column_index": 0, "card_index": 0, "hash": "abc.pdf",
	})
	b := parseBoard(t, dir)
	if got := b.Columns[0].Cards[0].Attachments; len(got) != 0 {
		t.Errorf("expected no attachments after remove, got %+v", got)
	}
}

func TestMCPMoveAttachment(t *testing.T) {
	srv, dir := setup(t)
	// Add attachment to card 0 in column 0.
	_ = callTool(t, srv, "card_add_attachment_ref", map[string]any{
		"board": "test-board", "column_index": 0, "card_index": 0,
		"hash": "abc.pdf", "name": "doc.pdf", "size": int64(1), "mime": "application/pdf",
	})
	// Move to card 1 in column 0.
	result := callTool(t, srv, "card_move_attachment", map[string]any{
		"board":     "test-board",
		"from_col":  0,
		"from_card": 0,
		"to_col":    0,
		"to_card":   1,
		"hash":      "abc.pdf",
	})
	text := resultText(t, result)
	if !strings.Contains(text, "moved") {
		t.Errorf("unexpected response: %q", text)
	}
	b := parseBoard(t, dir)
	if got := b.Columns[0].Cards[0].Attachments; len(got) != 0 {
		t.Errorf("expected no attachments on source card, got %+v", got)
	}
	if got := b.Columns[0].Cards[1].Attachments; len(got) != 1 || got[0].Hash != "abc.pdf" {
		t.Errorf("expected attachment on destination card, got %+v", got)
	}
}

func TestMCPRenameAttachment(t *testing.T) {
	srv, dir := setup(t)
	_ = callTool(t, srv, "card_add_attachment_ref", map[string]any{
		"board": "test-board", "column_index": 0, "card_index": 0,
		"hash": "abc.pdf", "name": "doc.pdf", "size": int64(1), "mime": "application/pdf",
	})
	result := callTool(t, srv, "card_rename_attachment", map[string]any{
		"board":        "test-board",
		"column_index": 0,
		"card_index":   0,
		"hash":         "abc.pdf",
		"new_name":     "renamed.pdf",
	})
	text := resultText(t, result)
	if !strings.Contains(text, "renamed") {
		t.Errorf("unexpected response: %q", text)
	}
	b := parseBoard(t, dir)
	if got := b.Columns[0].Cards[0].Attachments; len(got) != 1 || got[0].Name != "renamed.pdf" {
		t.Errorf("expected renamed attachment, got %+v", got)
	}
}

func TestMCPReorderAttachments(t *testing.T) {
	srv, dir := setup(t)
	// Add two attachments.
	_ = callTool(t, srv, "card_add_attachment_ref", map[string]any{
		"board": "test-board", "column_index": 0, "card_index": 0,
		"hash": "first.pdf", "name": "first.pdf", "size": int64(1), "mime": "application/pdf",
	})
	_ = callTool(t, srv, "card_add_attachment_ref", map[string]any{
		"board": "test-board", "column_index": 0, "card_index": 0,
		"hash": "second.pdf", "name": "second.pdf", "size": int64(2), "mime": "application/pdf",
	})
	// Reorder: second first.
	result := callTool(t, srv, "card_reorder_attachments", map[string]any{
		"board":           "test-board",
		"column_index":    0,
		"card_index":      0,
		"hashes_in_order": []string{"second.pdf", "first.pdf"},
	})
	text := resultText(t, result)
	if !strings.Contains(text, "reordered") {
		t.Errorf("unexpected response: %q", text)
	}
	b := parseBoard(t, dir)
	got := b.Columns[0].Cards[0].Attachments
	if len(got) != 2 || got[0].Hash != "second.pdf" || got[1].Hash != "first.pdf" {
		t.Errorf("unexpected order: %+v", got)
	}
}
