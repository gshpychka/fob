package store

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// Locations of the age tools; a build pins them with -X.
var (
	agePath         = "age"
	agePluginSEPath = "age-plugin-se"
)

// enclave seals with age, each secret to a key of its own in the Mac's Secure Enclave.
//
// https://github.com/remko/age-plugin-se
type enclave struct {
	// identities keeps the enclave's handle to each secret's key; a handle works on this Mac alone.
	identities files
}

// A key takes a touch for each use, and works while the enrolled fingers are those of its making.
const accessControl = "current-biometry"

func (e enclave) seal(ctx context.Context, name string, secret []byte) ([]byte, error) {
	// A new key for each secret written binds it to the fingers enrolled now.
	identity, err := run(ctx, nil, nil, agePluginSEPath, "keygen", "--access-control", accessControl)
	if err != nil {
		return nil, err
	}
	recipient, err := run(ctx, nil, identity, agePluginSEPath, "recipients")
	if err != nil {
		return nil, err
	}
	ciphertext, err := run(ctx, ageEnvironment(), secret, agePath, "--encrypt", "--recipient", strings.TrimSpace(string(recipient)))
	if err != nil {
		return nil, err
	}
	return ciphertext, e.identities.Set(ctx, name, identity)
}

func (e enclave) open(ctx context.Context, name string, ciphertext []byte) ([]byte, error) {
	identity, err := e.identities.path(name)
	if err != nil {
		return nil, err
	}
	return run(ctx, ageEnvironment(), ciphertext, agePath, "--decrypt", "--identity", identity)
}

// ageEnvironment is the environment in which age finds the plugin, which it resolves on PATH.
func ageEnvironment() []string {
	if !filepath.IsAbs(agePluginSEPath) {
		return nil
	}
	return append(os.Environ(), "PATH="+filepath.Dir(agePluginSEPath)+string(os.PathListSeparator)+os.Getenv("PATH"))
}
