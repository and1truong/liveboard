//go:build !darwin

package reminder

// SendSystemNotification is a no-op on non-macOS platforms.
func SendSystemNotification(_, _, _ string) error {
	return nil
}
