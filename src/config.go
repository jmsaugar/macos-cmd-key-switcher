package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config stores the last applied mode and manually learned keyboard classifications.
// Startup detection uses the classifications and can replace the saved mode.
type Config struct {
	// Type is the last successfully applied mapping: "mac" or "win".
	Type string `json:"type"`
	// KeyboardTypes maps decimal vendor:product or vendor:product:serial:<URL-escaped serial>
	// keys to "mac" or "win". Serial-based overrides take precedence over model overrides.
	KeyboardTypes map[string]string `json:"keyboard_types"`
}

// defaults returns a mac configuration with a new, empty keyboard preference map.
func defaults() Config {
	return Config{Type: "mac", KeyboardTypes: map[string]string{}}
}

// loadConfig reads and validates path, returning defaults if the file is absent.
// On error, the returned configuration may be partially populated and must not be used.
func loadConfig(path string) (Config, error) {
	c := defaults()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if err = json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	if c.Type != "mac" && c.Type != "win" {
		return c, fmt.Errorf("invalid keyboard type %q", c.Type)
	}
	for id, t := range c.KeyboardTypes {
		if t != "mac" && t != "win" {
			return c, fmt.Errorf("invalid keyboard override %s: %s", id, t)
		}
	}
	return c, nil
}

// saveConfig writes c as JSON to a temporary file in the destination directory
// and atomically renames it over path. It returns the first error encountered.
func saveConfig(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), "config-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
