// Package awsconfig keeps the AWS shared config file that gives every role a profile.
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
	"strings"

	"github.com/gshpychka/fob/internal/atomicfile"
	"github.com/gshpychka/fob/internal/roles"
)

// header opens a config that program wrote.
func header(program string) string {
	return fmt.Sprintf("# Written by %s, which replaces this file on every sync.\n", program)
}

// ErrForeign reports a config that the program did not write.
var ErrForeign = errors.New("foreign config")

// Read returns the names of the profiles in the config that program wrote at path.
func Read(path, program string) ([]string, error) {
	config, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(config) > 0 && !bytes.HasPrefix(config, []byte(header(program))) {
		return nil, ErrForeign
	}
	var names []string
	for line := range strings.Lines(string(config)) {
		if name, found := strings.CutPrefix(strings.TrimSpace(line), "[profile "); found {
			names = append(names, strings.TrimSuffix(name, "]"))
		}
	}
	return names, nil
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
