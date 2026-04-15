// Package renderer exposes the built default renderer bundle for embedding.
package renderer

import "embed"

// FS is the embedded default renderer bundle.
//
//go:embed all:dist
var FS embed.FS
