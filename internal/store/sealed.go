package store

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
)

// sealer encrypts secrets so that only this machine reads them back.
type sealer interface {
	seal(ctx context.Context, name string, secret []byte) ([]byte, error)
	open(ctx context.Context, name string, ciphertext []byte) ([]byte, error)
}

// sealed keeps each secret in a file in its sealed form.
type sealed struct {
	files  files
	sealer sealer
}

func (s sealed) Get(ctx context.Context, name string) ([]byte, error) {
	ciphertext, err := s.files.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	return s.sealer.open(ctx, name, ciphertext)
}

func (s sealed) Set(ctx context.Context, name string, secret []byte) error {
	ciphertext, err := s.sealer.seal(ctx, name, secret)
	if err != nil {
		return err
	}
	return s.files.Set(ctx, name, ciphertext)
}

// run feeds input to a sealing tool and returns what it prints.
func run(ctx context.Context, environment []string, input []byte, tool string, arguments ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, tool, arguments...)
	command.Env = environment
	command.Stdin = bytes.NewReader(input)
	var diagnostics bytes.Buffer
	command.Stderr = &diagnostics
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("%s: %w: %s", filepath.Base(tool), err, bytes.TrimSpace(diagnostics.Bytes()))
	}
	return output, nil
}
