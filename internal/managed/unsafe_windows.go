//go:build windows

package managed

import "unsafe"

func unsafePointer(value *uint16) unsafe.Pointer { return unsafe.Pointer(value) }
