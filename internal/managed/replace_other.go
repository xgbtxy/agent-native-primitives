//go:build !windows

package managed

import "os"

func replaceFile(source, destination string) error {
	return os.Rename(source, destination)
}
