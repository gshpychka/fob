package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Locations of the age tools; a build pins them with -X.
var (
	agePath         = "age"
	ageKeygenPath   = "age-keygen"
	agePluginSEPath = "age-plugin-se"
)

// keeper binds the key of a sealed store to this machine.
type keeper interface {
	// newKey makes a key, in the form that is kept at rest.
	newKey(ctx context.Context) ([]byte, error)
	// identity returns the age identity that a key gives on this machine.
	identity(ctx context.Context, key []byte) ([]byte, error)
}

// keyVariable names the environment variable that can hold the key of a sealed store.
const keyVariable = "FOB_STORE_KEY"

// storeKey returns the key in the environment, or the one kept in dir, which is made on first use.
func storeKey(ctx context.Context, keeper keeper, dir string) ([]byte, error) {
	if key := os.Getenv(keyVariable); key != "" {
		return []byte(key), nil
	}
	kept := files{dir: dir}
	key, err := kept.Get(ctx, "key")
	if !errors.Is(err, ErrNotFound) {
		return key, err
	}
	if key, err = keeper.newKey(ctx); err != nil {
		return nil, err
	}
	return key, kept.Set(ctx, "key", key)
}

// sealed keeps each secret in a file, encrypted with age to one identity.
//
// https://age-encryption.org
type sealed struct {
	files    files
	identity []byte
}

func (s sealed) Get(ctx context.Context, name string) ([]byte, error) {
	ciphertext, err := s.files.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	return s.age(ctx, "--decrypt", ciphertext)
}

func (s sealed) Set(ctx context.Context, name string, secret []byte) error {
	ciphertext, err := s.age(ctx, "--encrypt", secret)
	if err != nil {
		return err
	}
	return s.files.Set(ctx, name, ciphertext)
}

// age runs age over input in one mode; the identity is both the recipient and the key.
func (s sealed) age(ctx context.Context, mode string, input []byte) ([]byte, error) {
	// age reads an identity from a file, and a pipe is one that stays off the disk.
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	go func() {
		defer writer.Close()
		writer.Write(s.identity)
	}()
	age := exec.CommandContext(ctx, agePath, mode, "--identity", "/dev/fd/3")
	// The first extra file is the command's descriptor 3.
	age.ExtraFiles = []*os.File{reader}
	if filepath.IsAbs(agePluginSEPath) {
		// age resolves the plugin of an identity on PATH.
		age.Env = append(os.Environ(), "PATH="+filepath.Dir(agePluginSEPath)+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	return run(age, input)
}

// run feeds input to a command and returns what it prints.
func run(command *exec.Cmd, input []byte) ([]byte, error) {
	command.Stdin = bytes.NewReader(input)
	var diagnostics bytes.Buffer
	command.Stderr = &diagnostics
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("%s: %w: %s", filepath.Base(command.Path), err, bytes.TrimSpace(diagnostics.Bytes()))
	}
	return output, nil
}
