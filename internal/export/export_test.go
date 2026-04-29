package export

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/and1truong/liveboard/internal/attachments"
	"github.com/and1truong/liveboard/internal/workspace"
)

const boardAlpha = `---
version: 1
name: Product Roadmap
description: Q1 2026 planning
icon: "\U0001F680"
tags: [product, q1]
members: [alice, bob]
---

## Backlog

- [ ] Design new landing page #design
  assignee: alice
  priority: high
  due: 2026-04-15
  Needs to match brand guidelines.

- [ ] Set up CI pipeline #infra
  assignee: bob
  priority: medium

## In Progress

- [ ] Implement auth flow #backend
  assignee: alice
  priority: critical

## Done

- [x] Write project brief
  assignee: bob
  priority: low
`

const boardBeta = `---
version: 1
name: Bug Tracker
description: Active bugs
icon: "\U0001F41B"
tags: [bugs]
---

## Open

- [ ] Fix login timeout #backend
  priority: high

- [ ] CSS overflow on mobile #frontend
  priority: medium

## Resolved

- [x] Memory leak in worker
  priority: critical
`

func setupExportWorkspace(t *testing.T, boards map[string]string) *workspace.Workspace {
	t.Helper()
	dir := t.TempDir()
	for name, content := range boards {
		path := filepath.Join(dir, name+".md")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return workspace.Open(dir)
}

func readZipFiles(t *testing.T, data []byte) map[string]string {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("invalid zip: %v", err)
	}
	files := make(map[string]string, len(r.File))
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rc); err != nil {
			t.Fatal(err)
		}
		_ = rc.Close()
		files[f.Name] = buf.String()
	}
	return files
}

func TestRunToZip(t *testing.T) {
	ws := setupExportWorkspace(t, map[string]string{"roadmap": boardAlpha})

	data, err := RunToZip(ws, Options{})
	if err != nil {
		t.Fatal(err)
	}

	files := readZipFiles(t, data)

	if _, ok := files["index.html"]; !ok {
		t.Error("missing index.html in zip")
	}
	if _, ok := files["roadmap.html"]; !ok {
		t.Error("missing roadmap.html in zip")
	}

	// Verify index references the board
	idx := files["index.html"]
	if !strings.Contains(idx, "Product Roadmap") {
		t.Error("index.html should contain board name")
	}
	if !strings.Contains(idx, "roadmap.html") {
		t.Error("index.html should link to roadmap.html")
	}

	// Verify board HTML has expected content
	board := files["roadmap.html"]
	if !strings.Contains(board, "Product Roadmap") {
		t.Error("board html should contain board name")
	}
	if !strings.Contains(board, "Backlog") {
		t.Error("board html should contain column name")
	}
	if !strings.Contains(board, "Design new landing page") {
		t.Error("board html should contain card title")
	}
}

func TestWriteZipTo(t *testing.T) {
	ws := setupExportWorkspace(t, map[string]string{"roadmap": boardAlpha})

	var buf bytes.Buffer
	if err := WriteZipTo(&buf, ws, Options{}); err != nil {
		t.Fatal(err)
	}

	if buf.Len() == 0 {
		t.Fatal("WriteZipTo produced empty output")
	}

	files := readZipFiles(t, buf.Bytes())
	if _, ok := files["index.html"]; !ok {
		t.Error("missing index.html")
	}
	if _, ok := files["roadmap.html"]; !ok {
		t.Error("missing roadmap.html")
	}
}

func TestRun(t *testing.T) {
	ws := setupExportWorkspace(t, map[string]string{"roadmap": boardAlpha})
	outDir := filepath.Join(t.TempDir(), "export-output")

	if err := Run(ws, outDir, Options{}); err != nil {
		t.Fatal(err)
	}

	// Verify files on disk
	for _, name := range []string{"index.html", "roadmap.html"} {
		path := filepath.Join(outDir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected %s to exist: %v", name, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("%s is empty", name)
		}
	}

	// Verify content
	data, err := os.ReadFile(filepath.Join(outDir, "roadmap.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Product Roadmap") {
		t.Error("exported board should contain board name")
	}
}

func TestExportOptions(t *testing.T) {
	ws := setupExportWorkspace(t, map[string]string{"roadmap": boardAlpha})

	data, err := RunToZip(ws, Options{
		Theme:      "dark",
		ColorTheme: "ocean",
		SiteName:   "My Kanban",
	})
	if err != nil {
		t.Fatal(err)
	}

	files := readZipFiles(t, data)

	// Check theme attributes in index
	idx := files["index.html"]
	if !strings.Contains(idx, `data-theme="dark"`) {
		t.Error("index should have data-theme=dark")
	}
	if !strings.Contains(idx, `data-color-theme="ocean"`) {
		t.Error("index should have data-color-theme=ocean")
	}
	if !strings.Contains(idx, "My Kanban") {
		t.Error("index should have custom site name")
	}

	// Check board page too
	board := files["roadmap.html"]
	if !strings.Contains(board, `data-theme="dark"`) {
		t.Error("board should have data-theme=dark")
	}
	if !strings.Contains(board, `data-color-theme="ocean"`) {
		t.Error("board should have data-color-theme=ocean")
	}
	if !strings.Contains(board, "My Kanban") {
		t.Error("board should have custom site name")
	}
}

func TestExportOptions_SystemTheme(t *testing.T) {
	ws := setupExportWorkspace(t, map[string]string{"roadmap": boardAlpha})

	data, err := RunToZip(ws, Options{Theme: "system"})
	if err != nil {
		t.Fatal(err)
	}

	files := readZipFiles(t, data)
	idx := files["index.html"]
	if strings.Contains(idx, `data-theme="system"`) {
		t.Error("system theme should not produce data-theme attribute")
	}
}

func TestExportOptions_DefaultSiteName(t *testing.T) {
	ws := setupExportWorkspace(t, map[string]string{"roadmap": boardAlpha})

	data, err := RunToZip(ws, Options{})
	if err != nil {
		t.Fatal(err)
	}

	files := readZipFiles(t, data)
	if !strings.Contains(files["index.html"], "LiveBoard") {
		t.Error("default site name should be LiveBoard")
	}
}

func TestRunToZip_EmptyWorkspace(t *testing.T) {
	ws := setupExportWorkspace(t, map[string]string{})

	data, err := RunToZip(ws, Options{})
	if err != nil {
		t.Fatal(err)
	}

	files := readZipFiles(t, data)

	if _, ok := files["index.html"]; !ok {
		t.Error("empty workspace should still produce index.html")
	}

	// Should have only index.html
	if len(files) != 1 {
		t.Errorf("expected 1 file (index.html), got %d: %v", len(files), fileNames(files))
	}

	// Index should have 0 boards content
	idx := files["index.html"]
	if !strings.Contains(idx, "LiveBoard") {
		t.Error("index should still have site name")
	}
}

func TestRunToZip_MultipleBoards(t *testing.T) {
	ws := setupExportWorkspace(t, map[string]string{
		"roadmap": boardAlpha,
		"bugs":    boardBeta,
	})

	data, err := RunToZip(ws, Options{})
	if err != nil {
		t.Fatal(err)
	}

	files := readZipFiles(t, data)

	// Should have index + 2 board pages
	expected := []string{"index.html", "roadmap.html", "bugs.html"}
	for _, name := range expected {
		if _, ok := files[name]; !ok {
			t.Errorf("missing %s in zip", name)
		}
	}

	if len(files) != 3 {
		t.Errorf("expected 3 files, got %d: %v", len(files), fileNames(files))
	}

	// Index should reference both boards
	idx := files["index.html"]
	if !strings.Contains(idx, "Product Roadmap") {
		t.Error("index should contain Product Roadmap")
	}
	if !strings.Contains(idx, "Bug Tracker") {
		t.Error("index should contain Bug Tracker")
	}

	// Each board page should have its own content
	if !strings.Contains(files["roadmap.html"], "Design new landing page") {
		t.Error("roadmap.html should contain its cards")
	}
	if !strings.Contains(files["bugs.html"], "Fix login timeout") {
		t.Error("bugs.html should contain its cards")
	}

	// Board pages should have sidebar links to each other
	roadmap := files["roadmap.html"]
	if !strings.Contains(roadmap, "bugs.html") {
		t.Error("roadmap page should link to bugs page")
	}
}

func TestBuildSummaries(t *testing.T) {
	ws := setupExportWorkspace(t, map[string]string{
		"roadmap": boardAlpha,
		"bugs":    boardBeta,
	})

	boards, err := ws.ListBoards()
	if err != nil {
		t.Fatal(err)
	}

	summaries := buildSummaries(boards)
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}

	// Find roadmap summary
	var roadmap *boardSummary
	for i := range summaries {
		if summaries[i].Slug == "roadmap" {
			roadmap = &summaries[i]
			break
		}
	}
	if roadmap == nil {
		t.Fatal("roadmap summary not found")
		return
	}
	if roadmap.Name != "Product Roadmap" {
		t.Errorf("name = %q, want Product Roadmap", roadmap.Name)
	}
	if roadmap.ColumnCount != 3 {
		t.Errorf("columns = %d, want 3", roadmap.ColumnCount)
	}
	if roadmap.CardCount != 4 {
		t.Errorf("cards = %d, want 4", roadmap.CardCount)
	}
	if roadmap.DoneCount != 1 {
		t.Errorf("done = %d, want 1", roadmap.DoneCount)
	}
}

func fileNames(files map[string]string) []string {
	names := make([]string, 0, len(files))
	for k := range files {
		names = append(names, k)
	}
	return names
}

func zipNames(zr *zip.Reader) []string {
	out := make([]string, 0, len(zr.File))
	for _, f := range zr.File {
		out = append(out, f.Name)
	}
	return out
}

func TestExportMarkdownBundlesAttachments(t *testing.T) {
	dir := t.TempDir()
	store := attachments.NewStore(dir)
	desc, err := store.Put(bytes.NewReader([]byte("blob")), "x.txt")
	if err != nil {
		t.Fatal(err)
	}
	board := "---\nversion: 1\nname: B\n---\n\n## C\n\n- [ ] x\n  attachments: [{\"h\":\"" + desc.Hash + "\",\"n\":\"x.txt\",\"s\":4,\"m\":\"text/plain\"}]\n"
	if writeErr := os.WriteFile(filepath.Join(dir, "b.md"), []byte(board), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}

	var buf bytes.Buffer
	ws := workspace.Open(dir)
	if zipErr := WriteMarkdownZipToOpts(&buf, ws, Options{IncludeAttachments: true}); zipErr != nil {
		t.Fatal(zipErr)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	want := ".attachments/" + desc.Hash
	var found bool
	for _, f := range zr.File {
		if f.Name == want {
			found = true
		}
	}
	if !found {
		t.Errorf("attachment blob missing from zip; got entries: %v", zipNames(zr))
	}
}

func TestExportMarkdownSkipsAttachmentsWhenDisabled(t *testing.T) {
	dir := t.TempDir()
	store := attachments.NewStore(dir)
	desc, _ := store.Put(bytes.NewReader([]byte("blob")), "x.txt")
	board := "---\nversion: 1\nname: B\n---\n\n## C\n\n- [ ] x\n  attachments: [{\"h\":\"" + desc.Hash + "\",\"n\":\"x.txt\",\"s\":4,\"m\":\"text/plain\"}]\n"
	_ = os.WriteFile(filepath.Join(dir, "b.md"), []byte(board), 0o644)

	var buf bytes.Buffer
	ws := workspace.Open(dir)
	if err := WriteMarkdownZipToOpts(&buf, ws, Options{IncludeAttachments: false}); err != nil {
		t.Fatal(err)
	}
	zr, zrErr := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if zrErr != nil {
		t.Fatal(zrErr)
	}
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, ".attachments/") {
			t.Errorf("attachments included when disabled: %s", f.Name)
		}
	}
}

func TestExportHTMLBundlesAttachments(t *testing.T) {
	dir := t.TempDir()
	store := attachments.NewStore(dir)
	desc, _ := store.Put(bytes.NewReader([]byte("blob")), "x.txt")
	board := "---\nversion: 1\nname: B\n---\n\n## C\n\n- [ ] x\n  attachments: [{\"h\":\"" + desc.Hash + "\",\"n\":\"x.txt\",\"s\":4,\"m\":\"text/plain\"}]\n"
	_ = os.WriteFile(filepath.Join(dir, "b.md"), []byte(board), 0o644)

	var buf bytes.Buffer
	ws := workspace.Open(dir)
	if err := WriteZipTo(&buf, ws, Options{IncludeAttachments: true, SiteName: "X"}); err != nil {
		t.Fatal(err)
	}
	zr, zrErr := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if zrErr != nil {
		t.Fatal(zrErr)
	}
	want := ".attachments/" + desc.Hash
	var found bool
	for _, f := range zr.File {
		if f.Name == want {
			found = true
		}
	}
	if !found {
		t.Errorf("HTML export missing attachment blob; got entries: %v", zipNames(zr))
	}
}
