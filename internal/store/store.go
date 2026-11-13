// Package store keeps named secrets at rest.
package store

import (
	"context"
	"errors"
	"fmt"
	"os"
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
	// Age seals each file with age to the identity that the environment holds.
	Age Kind = "age"
	// SystemdCreds seals each file with age to an identity that the environment holds as a systemd credential.
	SystemdCreds Kind = "systemd-creds"
)

// keyVariable names the environment variable that holds the key of a sealed store.
const keyVariable = "FOB_STORE_KEY"

// DefaultKind is the kind that takes a key bound to the machine on this platform.
func DefaultKind() Kind {
	if runtime.GOOS == "darwin" {
		return Age
	}
	return SystemdCreds
}

// Open returns the store of one kind that lives in dir.
func Open(ctx context.Context, kind Kind, dir string) (Store, error) {
	secrets := files{dir: filepath.Join(dir, "secrets")}
	switch kind {
	case File:
		return secrets, nil
	case Age, SystemdCreds:
		identity := []byte(os.Getenv(keyVariable))
		if len(identity) == 0 {
			return nil, fmt.Errorf("the %s store takes its key from %s, which is unset", kind, keyVariable)
		}
		if kind == SystemdCreds {
			var err error
			if identity, err = unwrap(ctx, identity); err != nil {
				return nil, err
			}
		}
		return sealed{files: secrets, identity: identity}, nil
	}
	return nil, fmt.Errorf("unknown store %q: the stores are %s, %s and %s", kind, File, Age, SystemdCreds)
}
