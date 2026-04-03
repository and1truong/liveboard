//go:build !darwin

package reminder

// SendSystemNotification is a no-op on non-macOS platforms.
func SendSystemNotification(title, body, deepLinkURL string) error {
	return nil
}
