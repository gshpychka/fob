package store

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Locations of the age tools; a build pins them with -X.
var (
	agePath         = "age"
	agePluginSEPath = "age-plugin-se"
)

// enclave seals with age to a key in the Mac's Secure Enclave.
//
// https://github.com/remko/age-plugin-se
type enclave struct {
	// identityPath holds the enclave's handle to the key, which works on this Mac alone.
	identityPath string
}

// The key serves any process that reads the identity file, with no user presence check.
const accessControl = "none"

func (e enclave) seal(ctx context.Context, _ string, secret []byte) ([]byte, error) {
	if err := e.ensureIdentity(ctx); err != nil {
		return nil, err
	}
	recipient, err := run(ctx, nil, nil, agePluginSEPath, "recipients", "-i", e.identityPath)
	if err != nil {
		return nil, err
	}
	return run(ctx, ageEnvironment(), secret, agePath, "--encrypt", "--recipient", strings.TrimSpace(string(recipient)))
}

func (e enclave) open(ctx context.Context, _ string, ciphertext []byte) ([]byte, error) {
	return run(ctx, ageEnvironment(), ciphertext, agePath, "--decrypt", "--identity", e.identityPath)
}

// ensureIdentity generates the key on first use.
func (e enclave) ensureIdentity(ctx context.Context) error {
	if _, err := os.Stat(e.identityPath); !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	scratch, err := os.MkdirTemp(filepath.Dir(e.identityPath), "enclave-identity.*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(scratch)
	candidate := filepath.Join(scratch, "identity")
	if _, err := run(ctx, nil, nil, agePluginSEPath, "keygen", "--access-control", accessControl, "-o", candidate); err != nil {
		return err
	}
	if err := os.Chmod(candidate, 0o600); err != nil {
		return err
	}
	// A link fails when the identity exists, so processes racing through first use settle on one key.
	if err := os.Link(candidate, e.identityPath); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	return nil
}

// ageEnvironment is the environment in which age finds the plugin, which it resolves on PATH.
func ageEnvironment() []string {
	if !filepath.IsAbs(agePluginSEPath) {
		return nil
	}
	return append(os.Environ(), "PATH="+filepath.Dir(agePluginSEPath)+string(os.PathListSeparator)+os.Getenv("PATH"))
}
