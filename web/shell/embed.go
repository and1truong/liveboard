// Package shell exposes the built TS shell bundle for embedding in the Go server.
package shell

import "embed"

// FS holds the embedded shell dist directory produced by `make shell`.
//
//go:embed all:dist
var FS embed.FS
