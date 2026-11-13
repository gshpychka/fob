package store

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Locations of age and of its plugins; a build pins them with -X.
var (
	agePath        = "age"
	agePluginsPath = ""
)

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
	if agePluginsPath != "" {
		// age resolves the plugin of an identity on PATH.
		age.Env = append(os.Environ(), "PATH="+agePluginsPath+string(os.PathListSeparator)+os.Getenv("PATH"))
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
