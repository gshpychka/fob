// Package awsconfig writes the AWS shared config file that gives every role a profile.
//
// https://docs.aws.amazon.com/sdkref/latest/guide/file-format.html
package awsconfig

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/gshpychka/idc/internal/atomicfile"
	"github.com/gshpychka/idc/internal/roles"
)

// Write replaces the config at path with one in which each profile sources its credentials from program.
// The file is one program wrote before, or holds nothing yet.
func Write(path, program string, profiles []roles.Profile) error {
	header := fmt.Sprintf("# Written by %s, which replaces this file on every sync.\n", program)
	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if len(existing) > 0 && !bytes.HasPrefix(existing, []byte(header)) {
		return fmt.Errorf("%s holds a config %s did not write; point AWS_CONFIG_FILE at a file for %s to own", path, program, program)
	}
	config := bytes.NewBufferString(header)
	for _, profile := range profiles {
		fmt.Fprintf(config, "\n[profile %s]\ncredential_process = %s creds %s %s\n",
			profile.Name, program, profile.Role.AccountID, profile.Role.Name)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return atomicfile.Write(path, config.Bytes())
}
