package store

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gshpychka/fob/internal/atomicfile"
)

// files keeps each secret in a file named after it.
type files struct {
	dir string
}

func (f files) Get(_ context.Context, name string) ([]byte, error) {
	path, err := f.path(name)
	if err != nil {
		return nil, err
	}
	secret, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotFound
	}
	return secret, err
}

func (f files) Set(_ context.Context, name string, secret []byte) error {
	path, err := f.path(name)
	if err != nil {
		return err
	}
	return atomicfile.Write(path, secret)
}

func (f files) path(name string) (string, error) {
	if !filepath.IsLocal(name) || strings.ContainsRune(name, filepath.Separator) {
		return "", fmt.Errorf("secret name %q is not a file name", name)
	}
	return filepath.Join(f.dir, name), nil
}
