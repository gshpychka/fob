// Package atomicfile writes files that readers see whole.
package atomicfile

import (
	"os"
	"path/filepath"
)

// Write puts data at path, readable by its owner.
func Write(path string, data []byte) error {
	// A rename swaps the file in one step, so a concurrent reader gets the old content or the new.
	temporary, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), path)
}
