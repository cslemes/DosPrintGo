//go:build windows

package printer

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	winspool            = windows.NewLazySystemDLL("winspool.drv")
	procOpenPrinter     = winspool.NewProc("OpenPrinterW")
	procClosePrinter    = winspool.NewProc("ClosePrinter")
	procStartDocPrinter = winspool.NewProc("StartDocPrinterW")
	procEndDocPrinter   = winspool.NewProc("EndDocPrinter")
	procWritePrinter    = winspool.NewProc("WritePrinter")
)

type docInfo1 struct {
	pDocName    *uint16
	pOutputFile *uint16
	pDatatype   *uint16
}

// SendWindowsSpooler sends raw bytes to an installed Windows printer using RAW data type (no driver conversion).
func SendWindowsSpooler(printerName string, docName string, data []byte) error {
	if len(data) == 0 {
		return nil
	}

	pPrinterName, err := windows.UTF16PtrFromString(printerName)
	if err != nil {
		return err
	}

	var hPrinter windows.Handle
	r1, _, lastErr := procOpenPrinter.Call(
		uintptr(unsafe.Pointer(pPrinterName)),
		uintptr(unsafe.Pointer(&hPrinter)),
		0,
	)
	if r1 == 0 {
		return fmt.Errorf("OpenPrinter falhou para '%s': %w", printerName, lastErr)
	}
	defer procClosePrinter.Call(uintptr(hPrinter))

	pDoc, _ := windows.UTF16PtrFromString(docName)
	pRaw, _ := windows.UTF16PtrFromString("RAW")

	doc := docInfo1{
		pDocName:    pDoc,
		pOutputFile: nil,
		pDatatype:   pRaw,
	}

	jobID, _, lastErr := procStartDocPrinter.Call(
		uintptr(hPrinter),
		1,
		uintptr(unsafe.Pointer(&doc)),
	)
	if jobID == 0 {
		return fmt.Errorf("StartDocPrinter falhou para '%s': %w", printerName, lastErr)
	}
	defer procEndDocPrinter.Call(uintptr(hPrinter))

	var written uint32
	r1, _, lastErr = procWritePrinter.Call(
		uintptr(hPrinter),
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
		uintptr(unsafe.Pointer(&written)),
	)
	if r1 == 0 {
		return fmt.Errorf("WritePrinter falhou para '%s': %w", printerName, lastErr)
	}

	return nil
}
