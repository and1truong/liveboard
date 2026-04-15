// Package renderer exposes the built default renderer bundle for embedding.
package renderer

import "embed"

//go:embed all:dist
var FS embed.FS
