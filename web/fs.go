// Package web embeds static frontend assets.
package web

import "embed"

// FS embeds the CSS, JS, and image static assets.
//
//go:embed css js img
var FS embed.FS
