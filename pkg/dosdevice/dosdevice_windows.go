//go:build windows

package dosdevice

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32         = windows.NewLazySystemDLL("kernel32.dll")
	procDefineDosDev = kernel32.NewProc("DefineDosDeviceW")
)

const (
	DDD_RAW_TARGET_PATH       = 0x00000001
	DDD_REMOVE_DEFINITION     = 0x00000002
	DDD_EXACT_MATCH_ON_REMOVE = 0x00000004
	DDD_NO_BROADCAST_SYSTEM   = 0x00000008
)

// RedirectLPT binds a DOS device name (e.g. "LPT1") to targetFilePath in the Windows Object Manager.
func RedirectLPT(portName string, targetFilePath string) error {
	pPort, err := windows.UTF16PtrFromString(portName)
	if err != nil {
		return err
	}
	pPath, err := windows.UTF16PtrFromString(targetFilePath)
	if err != nil {
		return err
	}

	// First, try removing any stale previous definition for this device
	procDefineDosDev.Call(
		DDD_REMOVE_DEFINITION,
		uintptr(unsafe.Pointer(pPort)),
		0,
	)

	// DefineDosDeviceW with flags=0 maps portName to targetFilePath
	r1, _, lastErr := procDefineDosDev.Call(
		0,
		uintptr(unsafe.Pointer(pPort)),
		uintptr(unsafe.Pointer(pPath)),
	)
	if r1 == 0 {
		return fmt.Errorf("DefineDosDeviceW falhou para %s -> %s: %w", portName, targetFilePath, lastErr)
	}

	return nil
}

// RemoveLPT removes the redirection for portName.
func RemoveLPT(portName string, targetFilePath string) error {
	pPort, err := windows.UTF16PtrFromString(portName)
	if err != nil {
		return err
	}
	pPath, err := windows.UTF16PtrFromString(targetFilePath)
	if err != nil {
		return err
	}

	r1, _, lastErr := procDefineDosDev.Call(
		DDD_REMOVE_DEFINITION|DDD_EXACT_MATCH_ON_REMOVE,
		uintptr(unsafe.Pointer(pPort)),
		uintptr(unsafe.Pointer(pPath)),
	)
	if r1 == 0 {
		return fmt.Errorf("Remove DefineDosDeviceW falhou para %s: %w", portName, lastErr)
	}

	return nil
}
