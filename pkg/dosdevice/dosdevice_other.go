//go:build !windows

package dosdevice

// RedirectLPT is a no-op on non-Windows platforms
func RedirectLPT(portName string, targetFilePath string) error {
	return nil
}

// RemoveLPT is a no-op on non-Windows platforms
func RemoveLPT(portName string, targetFilePath string) error {
	return nil
}
