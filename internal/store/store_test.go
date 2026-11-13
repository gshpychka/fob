package store_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/gshpychka/fob/internal/store"
)

// This test runs where age is installed.
func TestSealedStoreOpensWithItsKey(t *testing.T) {
	dir := t.TempDir()
	// open returns the store in dir under a new key.
	open := func() store.Store {
		identity, err := exec.Command("age-keygen").Output()
		if err != nil {
			t.Skip("age-keygen is unavailable")
		}
		t.Setenv("FOB_STORE_KEY", string(identity))
		secrets, err := store.Open(t.Context(), store.Age, dir)
		if err != nil {
			t.Fatal(err)
		}
		return secrets
	}
	const name, secret = "session", "the secret"
	secrets := open()
	if err := secrets.Set(t.Context(), name, []byte(secret)); err != nil {
		t.Fatal(err)
	}
	if stored, err := secrets.Get(t.Context(), name); err != nil || string(stored) != secret {
		t.Errorf("stored %q, %v", stored, err)
	}
	if onDisk, err := os.ReadFile(filepath.Join(dir, "secrets", name)); err != nil || bytes.Contains(onDisk, []byte(secret)) {
		t.Errorf("the file is unreadable or holds the secret in the clear: %v", err)
	}
	if _, err := open().Get(t.Context(), name); err == nil {
		t.Error("the secret opened under another key")
	}
}
