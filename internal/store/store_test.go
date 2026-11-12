package store_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/gshpychka/fob/internal/store"
)

// This test runs on a machine whose systemd serves user-scoped credentials.
func TestSystemdCredsSealsToTheSecretsName(t *testing.T) {
	if exec.Command("systemd-creds", "--user", "--name=probe", "encrypt", "/dev/null", "-").Run() != nil {
		t.Skip("systemd-creds --user is unavailable")
	}
	ctx := context.Background()
	dir := t.TempDir()
	secrets, err := store.Open(store.SystemdCreds, dir)
	if err != nil {
		t.Fatal(err)
	}
	// Role names carry these characters.
	const name, secret = "role.123456789012.Power+User=,.@-Access", "the secret"
	if err := secrets.Set(ctx, name, []byte(secret)); err != nil {
		t.Fatal(err)
	}
	if stored, err := secrets.Get(ctx, name); err != nil || string(stored) != secret {
		t.Errorf("stored %q, %v", stored, err)
	}
	sealed := filepath.Join(dir, "secrets", name)
	if onDisk, err := os.ReadFile(sealed); err != nil || bytes.Contains(onDisk, []byte(secret)) {
		t.Errorf("the file is unreadable or holds the secret in the clear: %v", err)
	}
	if err := os.Rename(sealed, filepath.Join(dir, "secrets", "other")); err != nil {
		t.Fatal(err)
	}
	if _, err := secrets.Get(ctx, "other"); err == nil {
		t.Error("a sealed file opened under another name")
	}
}
