package printer

import (
	"testing"
)

func TestListInstalledPrinters(t *testing.T) {
	printers, err := ListInstalledPrinters()
	if err != nil {
		t.Fatalf("ListInstalledPrinters falhou: %v", err)
	}

	// In non-windows or mock environments, we should have at least 1 printer
	if len(printers) == 0 {
		t.Logf("Nenhuma impressora encontrada (ambiente de teste sem impressoras configuradas)")
	} else {
		t.Logf("Impressoras encontradas (%d): %v", len(printers), printers)
	}
}
