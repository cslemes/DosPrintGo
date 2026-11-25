//go:build !windows

package printer

// ListInstalledPrinters returns a mock list of printers on non-Windows platforms for testing and development.
func ListInstalledPrinters() ([]string, error) {
	return []string{
		"Zebra ZD220 (RAW)",
		"Argox OS-214 Plus (RAW)",
		"Epson TM-T20X (Cupom)",
		"Microsoft Print to PDF",
	}, nil
}
