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

	"github.com/gshpychka/fob/internal/atomicfile"
	"github.com/gshpychka/fob/internal/roles"
)

// header opens a config that program wrote.
func header(program string) string {
	return fmt.Sprintf("# Written by %s, which replaces this file on every sync.\n", program)
}

// Foreign reports whether the file at path holds a config that program did not write.
func Foreign(path, program string) (bool, error) {
	existing, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return len(existing) > 0 && !bytes.HasPrefix(existing, []byte(header(program))), err
}

// Write replaces the config at path with one in which each profile sources its credentials from program.
func Write(path, program string, profiles []roles.Profile) error {
	config := bytes.NewBufferString(header(program))
	for _, profile := range profiles {
		fmt.Fprintf(config, "\n[profile %s]\ncredential_process = %s creds %s %s\n",
			profile.Name, program, profile.Role.AccountID, profile.Role.Name)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return atomicfile.Write(path, config.Bytes())
}
