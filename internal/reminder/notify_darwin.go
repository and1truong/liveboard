//go:build darwin

package reminder

import (
	"fmt"
	"os/exec"
	"strings"
)

// SendSystemNotification sends a macOS notification via osascript.
func SendSystemNotification(title, body, deepLinkURL string) error {
	// Escape quotes for AppleScript
	title = strings.ReplaceAll(title, `"`, `\"`)
	body = strings.ReplaceAll(body, `"`, `\"`)

	script := fmt.Sprintf(`display notification "%s" with title "%s" sound name "default"`, body, title)

	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}
