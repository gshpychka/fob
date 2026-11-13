// Package store keeps named secrets at rest.
package store

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
)

// ErrNotFound reports a name without a secret stored under it.
var ErrNotFound = errors.New("secret not found")

// Store holds secrets by name.
type Store interface {
	// Get returns the secret stored under name, or ErrNotFound.
	Get(ctx context.Context, name string) ([]byte, error)
	Set(ctx context.Context, name string, secret []byte) error
}

// Kind names a storage backend.
type Kind string

const (
	// File keeps each secret in a file only its owner can read.
	File Kind = "file"
	// SecureEnclave seals each file with age to a key in the Mac's Secure Enclave.
	SecureEnclave Kind = "secure-enclave"
	// SystemdCreds seals each file with age to a key that systemd seals for the machine and the user.
	SystemdCreds Kind = "systemd-creds"
)

// DefaultKind is the backend that binds secrets to the machine on this platform.
func DefaultKind() Kind {
	if runtime.GOOS == "darwin" {
		return SecureEnclave
	}
	return SystemdCreds
}

// Open returns the store of one kind that lives in dir.
func Open(ctx context.Context, kind Kind, dir string) (Store, error) {
	secrets := files{dir: filepath.Join(dir, "secrets")}
	var keeper keeper
	switch kind {
	case File:
		return secrets, nil
	case SecureEnclave:
		keeper = enclave{}
	case SystemdCreds:
		keeper = systemdCreds{}
	default:
		return nil, fmt.Errorf("unknown store %q: the stores are %s, %s and %s", kind, File, SecureEnclave, SystemdCreds)
	}
	key, err := storeKey(ctx, keeper, dir)
	if err != nil {
		return nil, err
	}
	identity, err := keeper.identity(ctx, key)
	if err != nil {
		return nil, err
	}
	return sealed{files: secrets, identity: identity}, nil
}
