//go:build !windows

package printer

import "fmt"

// SendWindowsSpooler is a stub on non-Windows platforms
func SendWindowsSpooler(printerName string, docName string, data []byte) error {
	return fmt.Errorf("Windows Spooler não está disponível em plataformas não-Windows")
}
