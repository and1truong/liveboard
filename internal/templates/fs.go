// Package templates embeds HTML template files.
package templates

import "embed"

// FS embeds the HTML template files.
//
//go:embed *.html
var FS embed.FS
