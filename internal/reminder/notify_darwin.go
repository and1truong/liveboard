//go:build darwin

package reminder

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// SendSystemNotification sends a macOS notification via osascript.
func SendSystemNotification(title, body, _ string) error {
	// Escape quotes for AppleScript
	title = strings.ReplaceAll(title, `"`, `\"`)
	body = strings.ReplaceAll(body, `"`, `\"`)

	script := fmt.Sprintf(`display notification "%s" with title "%s" sound name "default"`, body, title)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	return cmd.Run()
}
