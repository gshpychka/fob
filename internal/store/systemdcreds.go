package store

import "context"

// systemdCreds seals with the machine's systemd credential key, scoped to the calling user.
//
// https://www.freedesktop.org/software/systemd/man/latest/systemd-creds.html
type systemdCreds struct{}

// A sealed credential carries its name, which ties a file to the secret it was written for.
func (systemdCreds) seal(ctx context.Context, name string, secret []byte) ([]byte, error) {
	return run(ctx, nil, secret, "systemd-creds", "--user", "--name="+name, "encrypt", "-", "-")
}

func (systemdCreds) open(ctx context.Context, name string, ciphertext []byte) ([]byte, error) {
	return run(ctx, nil, ciphertext, "systemd-creds", "--user", "--name="+name, "decrypt", "-", "-")
}
