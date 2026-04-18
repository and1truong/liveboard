package templates

import (
	"fmt"
	"html/template"
	"regexp"
)

// BoardIconColors lists the palette keys mirrored in iconColors.ts.
// Light-theme only (the static export doesn't switch themes yet).
var boardIconPalette = map[string][2]string{
	"slate":   {"#e2e8f0", "#334155"},
	"red":     {"#fee2e2", "#b91c1c"},
	"orange":  {"#ffedd5", "#c2410c"},
	"amber":   {"#fef3c7", "#b45309"},
	"yellow":  {"#fef9c3", "#a16207"},
	"lime":    {"#ecfccb", "#4d7c0f"},
	"green":   {"#dcfce7", "#15803d"},
	"teal":    {"#ccfbf1", "#0f766e"},
	"cyan":    {"#cffafe", "#0e7490"},
	"blue":    {"#dbeafe", "#1d4ed8"},
	"indigo":  {"#e0e7ff", "#4338ca"},
	"violet":  {"#ede9fe", "#6d28d9"},
	"fuchsia": {"#fae8ff", "#a21caf"},
	"pink":    {"#fce7f3", "#be185d"},
}

// IsEmojiIcon mirrors isEmojiIcon in web/renderer/default/src/icons/boardIcons.ts.
// Returns true when `icon` should be rendered as inline text (emoji/unicode)
// rather than looked up in BoardIconSVGs.
func IsEmojiIcon(icon string) bool {
	if icon == "" {
		return false
	}
	if _, ok := BoardIconSVGs[icon]; ok {
		return false
	}
	if len([]rune(icon)) > 4 {
		return false
	}
	return !latinRE.MatchString(icon)
}

var latinRE = regexp.MustCompile(`[a-zA-Z]`)

// BoardIconChip renders a tinted chip containing either the emoji or the SVG
// for a given icon slug. Empty icon renders a slate "list" chip.
func BoardIconChip(icon, colorKey string) template.HTML {
	pair, ok := boardIconPalette[colorKey]
	if !ok {
		pair = boardIconPalette["slate"]
	}
	style := fmt.Sprintf(
		"display:inline-flex;align-items:center;justify-content:center;width:20px;height:20px;border-radius:6px;background:%s;color:%s;flex-shrink:0;",
		pair[0], pair[1],
	)
	var inner string
	switch {
	case IsEmojiIcon(icon):
		style += "font-size:14px;background:transparent;"
		inner = template.HTMLEscapeString(icon)
	case icon != "":
		if svg, svgOK := BoardIconSVGs[icon]; svgOK {
			inner = string(svg)
		} else {
			inner = string(BoardIconSVGs["list"])
		}
	default:
		inner = string(BoardIconSVGs["list"])
	}
	return template.HTML(fmt.Sprintf(`<span class="export-bicon" style="%s" aria-hidden>%s</span>`, style, inner)) //nolint:gosec
}
