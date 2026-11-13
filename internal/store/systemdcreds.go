package store

import (
	"context"
	"os/exec"
)

// systemdCreds keeps the key as a credential that systemd seals for this machine and user.
//
// https://www.freedesktop.org/software/systemd/man/latest/systemd-creds.html
type systemdCreds struct{}

func (s systemdCreds) newKey(ctx context.Context) ([]byte, error) {
	identity, err := run(exec.CommandContext(ctx, ageKeygenPath), nil)
	if err != nil {
		return nil, err
	}
	return s.creds(ctx, "encrypt", identity)
}

func (s systemdCreds) identity(ctx context.Context, key []byte) ([]byte, error) {
	return s.creds(ctx, "decrypt", key)
}

// creds runs systemd-creds over input; the credential carries the name of the variable that can hold it.
func (systemdCreds) creds(ctx context.Context, verb string, input []byte) ([]byte, error) {
	return run(exec.CommandContext(ctx, "systemd-creds", "--user", "--name="+keyVariable, verb, "-", "-"), input)
}
