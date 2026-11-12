// Package config is the program's settings file.
package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/gshpychka/idc/internal/atomicfile"
	"github.com/gshpychka/idc/internal/session"
	"github.com/gshpychka/idc/internal/store"
)

// Settings are what the user tells the program once.
type Settings struct {
	// The IAM Identity Center instance to sign in to.
	session.Instance
	// Store names where secrets rest; left empty, it is the platform's default.
	Store store.Kind `json:"store,omitempty"`
}

// Load reads the settings at path; a missing file holds none.
func Load(path string) (Settings, error) {
	var settings Settings
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return settings, nil
	}
	if err != nil {
		return settings, err
	}
	return settings, json.Unmarshal(data, &settings)
}

// Save writes the settings to path.
func Save(path string, settings Settings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return atomicfile.Write(path, append(data, '\n'))
}
