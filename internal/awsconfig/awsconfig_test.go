package awsconfig_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gshpychka/idc/internal/awsconfig"
)

func TestWriteLeavesAForeignConfigAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	const foreign = "[default]\nregion = eu-west-1\n"
	if err := os.WriteFile(path, []byte(foreign), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := awsconfig.Write(path, "idc", nil); err == nil {
		t.Error("a config written by hand was replaced")
	}
	if kept, _ := os.ReadFile(path); string(kept) != foreign {
		t.Errorf("config is now %q", kept)
	}
}
