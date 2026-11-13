package store

import (
	"context"
	"os/exec"
)

// unwrap returns the age identity inside a credential that systemd sealed for this machine and user.
//
// https://www.freedesktop.org/software/systemd/man/latest/systemd-creds.html
func unwrap(ctx context.Context, credential []byte) ([]byte, error) {
	// The credential is sealed under the name of the variable that holds it.
	return run(exec.CommandContext(ctx, "systemd-creds", "--user", "--name="+keyVariable, "decrypt", "-", "-"), credential)
}
