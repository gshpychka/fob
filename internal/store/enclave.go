package store

import (
	"context"
	"os/exec"
)

// enclave keeps the key in the Mac's Secure Enclave; the key at rest is the enclave's handle to it.
//
// https://github.com/remko/age-plugin-se
type enclave struct{}

func (enclave) newKey(ctx context.Context) ([]byte, error) {
	return run(exec.CommandContext(ctx, agePluginSEPath, "keygen", "--access-control", "none"), nil)
}

// The handle is an age identity, and age has the enclave use the key.
func (enclave) identity(_ context.Context, key []byte) ([]byte, error) {
	return key, nil
}
