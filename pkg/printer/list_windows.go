//go:build windows

package printer

import (
	"fmt"
	"sort"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	printerEnumDefault     = 0x00000001
	printerEnumLocal       = 0x00000002
	printerEnumConnections = 0x00000004
)

var (
	procEnumPrinters = winspool.NewProc("EnumPrintersW")
)

type printerInfo4 struct {
	pPrinterName *uint16
	pServerName  *uint16
	attributes   uint32
}

type printerInfo1 struct {
	flags        uint32
	pDescription *uint16
	pName        *uint16
	pComment     *uint16
}

// ListInstalledPrinters lists all local, network, and connected printers installed in Windows.
func ListInstalledPrinters() ([]string, error) {
	flags := uint32(printerEnumLocal | printerEnumConnections)

	// Try Level 4 first (fast, returns local & connected printers)
	printers, err := enumPrintersLevel4(flags)
	if err == nil && len(printers) > 0 {
		sort.Strings(printers)
		return printers, nil
	}

	// Fallback to Level 1 (universal compatibility)
	printersLevel1, errLevel1 := enumPrintersLevel1(flags)
	if errLevel1 == nil && len(printersLevel1) > 0 {
		sort.Strings(printersLevel1)
		return printersLevel1, nil
	}

	if err != nil {
		return nil, err
	}
	return printers, nil
}

func enumPrintersLevel4(flags uint32) ([]string, error) {
	var pcbNeeded, pcReturned uint32

	// Query needed buffer size
	procEnumPrinters.Call(
		uintptr(flags),
		0,
		4,
		0,
		0,
		uintptr(unsafe.Pointer(&pcbNeeded)),
		uintptr(unsafe.Pointer(&pcReturned)),
	)

	if pcbNeeded == 0 {
		return []string{}, nil
	}

	buf := make([]byte, pcbNeeded)
	r1, _, lastErr := procEnumPrinters.Call(
		uintptr(flags),
		0,
		4,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(pcbNeeded),
		uintptr(unsafe.Pointer(&pcbNeeded)),
		uintptr(unsafe.Pointer(&pcReturned)),
	)
	if r1 == 0 {
		return nil, fmt.Errorf("EnumPrintersW Level 4 falhou: %w", lastErr)
	}

	printers := make([]string, 0, pcReturned)
	if pcReturned > 0 {
		infoSlice := (*[1 << 20]printerInfo4)(unsafe.Pointer(&buf[0]))[:pcReturned:pcReturned]
		for _, info := range infoSlice {
			if info.pPrinterName != nil {
				name := windows.UTF16PtrToString(info.pPrinterName)
				if name != "" {
					printers = append(printers, name)
				}
			}
		}
	}
	return printers, nil
}

func enumPrintersLevel1(flags uint32) ([]string, error) {
	var pcbNeeded, pcReturned uint32

	procEnumPrinters.Call(
		uintptr(flags),
		0,
		1,
		0,
		0,
		uintptr(unsafe.Pointer(&pcbNeeded)),
		uintptr(unsafe.Pointer(&pcReturned)),
	)

	if pcbNeeded == 0 {
		return []string{}, nil
	}

	buf := make([]byte, pcbNeeded)
	r1, _, lastErr := procEnumPrinters.Call(
		uintptr(flags),
		0,
		1,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(pcbNeeded),
		uintptr(unsafe.Pointer(&pcbNeeded)),
		uintptr(unsafe.Pointer(&pcReturned)),
	)
	if r1 == 0 {
		return nil, fmt.Errorf("EnumPrintersW Level 1 falhou: %w", lastErr)
	}

	printers := make([]string, 0, pcReturned)
	if pcReturned > 0 {
		infoSlice := (*[1 << 20]printerInfo1)(unsafe.Pointer(&buf[0]))[:pcReturned:pcReturned]
		for _, info := range infoSlice {
			if info.pName != nil {
				name := windows.UTF16PtrToString(info.pName)
				if name != "" {
					printers = append(printers, name)
				}
			}
		}
	}
	return printers, nil
}
