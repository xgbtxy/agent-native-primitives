//go:build windows

package managed

import (
	"fmt"
	"syscall"
)

const (
	moveFileReplaceExisting = 0x1
	moveFileWriteThrough    = 0x8
)

var moveFileEx = syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")

func replaceFile(source, destination string) error {
	from, err := syscall.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	to, err := syscall.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}
	result, _, callErr := moveFileEx.Call(uintptr(unsafePointer(from)), uintptr(unsafePointer(to)), moveFileReplaceExisting|moveFileWriteThrough)
	if result == 0 {
		return fmt.Errorf("replace manifest: %w", callErr)
	}
	return nil
}
