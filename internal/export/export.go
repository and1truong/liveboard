// Package export renders a workspace to static HTML files.
package export

import (
	"archive/zip"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"

	tmplfs "github.com/and1truong/liveboard/internal/templates"
	"github.com/and1truong/liveboard/internal/workspace"
	"github.com/and1truong/liveboard/pkg/models"
)

var mdRenderer = goldmark.New(
	goldmark.WithExtensions(extension.Linkify),
	goldmark.WithRendererOptions(html.WithHardWraps(), html.WithXHTML()),
)

// Options controls export rendering.
type Options struct {
	Theme      string
	ColorTheme string
	SiteName   string
}

type boardSummary struct {
	Name        string
	Slug        string
	Description string
	Icon        string
	Tags        []string
	CardCount   int
	DoneCount   int
	ColumnCount int
}

type indexModel struct {
	SiteName   string
	Theme      string
	ColorTheme string
	Boards     []boardSummary
}

type boardModel struct {
	Board      models.Board
	Slug       string
	Boards     []boardSummary
	SiteName   string
	Theme      string
	ColorTheme string
}

// renderFile is a callback that receives a filename and rendered content.
type renderFile func(name string, data []byte) error

// render builds all HTML pages and calls emit for each one.
func render(ws *workspace.Workspace, opts Options, emit renderFile) error {
	if opts.SiteName == "" {
		opts.SiteName = "LiveBoard"
	}

	boards, err := ws.ListBoards()
	if err != nil {
		return fmt.Errorf("listing boards: %w", err)
	}

	fm := template.FuncMap{
		"md": func(s string) template.HTML {
			var buf bytes.Buffer
			if err := mdRenderer.Convert([]byte(s), &buf); err != nil {
				return template.HTML(template.HTMLEscapeString(s))
			}
			return template.HTML(buf.String()) //nolint:gosec
		},
	}

	exportTemplates := []string{"export_styles.html"}
	indexTpl, err := template.New("export_index.html").Funcs(fm).ParseFS(tmplfs.FS, append(exportTemplates, "export_index.html")...)
	if err != nil {
		return fmt.Errorf("parsing index template: %w", err)
	}
	boardTpl, err := template.New("export_board.html").Funcs(fm).ParseFS(tmplfs.FS, append(exportTemplates, "export_board.html")...)
	if err != nil {
		return fmt.Errorf("parsing board template: %w", err)
	}

	// Build summaries
	var summaries []boardSummary
	for _, b := range boards {
		slug := strings.TrimSuffix(filepath.Base(b.FilePath), ".md")
		cards, done := 0, 0
		for _, col := range b.Columns {
			cards += len(col.Cards)
			for _, c := range col.Cards {
				if c.Completed {
					done++
				}
			}
		}
		summaries = append(summaries, boardSummary{
			Name:        b.Name,
			Slug:        slug,
			Description: b.Description,
			Icon:        b.Icon,
			Tags:        b.Tags,
			CardCount:   cards,
			DoneCount:   done,
			ColumnCount: len(b.Columns),
		})
	}

	// Resolve theme value for templates: "system" means no data-theme attribute
	themeAttr := opts.Theme
	if themeAttr == "system" {
		themeAttr = ""
	}

	// Render each board
	for i, b := range boards {
		var buf bytes.Buffer
		if err := boardTpl.Execute(&buf, boardModel{
			Board:      b,
			Slug:       summaries[i].Slug,
			Boards:     summaries,
			SiteName:   opts.SiteName,
			Theme:      themeAttr,
			ColorTheme: opts.ColorTheme,
		}); err != nil {
			return fmt.Errorf("rendering board %q: %w", b.Name, err)
		}
		if err := emit(summaries[i].Slug+".html", buf.Bytes()); err != nil {
			return err
		}
	}

	// Render index
	var buf bytes.Buffer
	if err := indexTpl.Execute(&buf, indexModel{
		SiteName:   opts.SiteName,
		Theme:      themeAttr,
		ColorTheme: opts.ColorTheme,
		Boards:     summaries,
	}); err != nil {
		return fmt.Errorf("rendering index: %w", err)
	}
	return emit("index.html", buf.Bytes())
}

// RunToZip renders all boards to an in-memory ZIP archive and returns the bytes.
func RunToZip(ws *workspace.Workspace, opts Options) ([]byte, error) {
	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)

	err := render(ws, opts, func(name string, data []byte) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return zipBuf.Bytes(), nil
}

// Run exports all boards in the workspace to outputDir as static HTML.
func Run(ws *workspace.Workspace, outputDir string, opts Options) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	count := 0
	err := render(ws, opts, func(name string, data []byte) error {
		outPath := filepath.Join(outputDir, name)
		if err := os.WriteFile(outPath, data, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", outPath, err)
		}
		fmt.Printf("  %s\n", outPath)
		count++
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Printf("Exported %d files to %s\n", count, outputDir)
	return nil
}

// WriteZipTo renders all boards and writes the ZIP archive to w.
func WriteZipTo(w io.Writer, ws *workspace.Workspace, opts Options) error {
	data, err := RunToZip(ws, opts)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
