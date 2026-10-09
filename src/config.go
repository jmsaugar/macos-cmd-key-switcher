package main

import (
	"encoding/json"
	"fmt"
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

// preferencesStore transports a configuration snapshot to and from native storage.
// JSON is used only across the bridge; the native backend stores a dictionary.
type preferencesStore interface {
	read() ([]byte, error)
	write([]byte) error
	clear() error
}

// loadConfig reads and validates preferences, returning defaults when none exist.
// On error, the returned configuration may be partially populated and must not be used.
func loadConfig(preferences preferencesStore) (Config, error) {
	c := defaults()
	data, err := preferences.read()
	if err != nil {
		return c, err
	}
	if len(data) == 0 {
		return c, nil
	}
	if err = json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	if c.KeyboardTypes == nil {
		c.KeyboardTypes = map[string]string{}
	}
	return c, validateConfig(c)
}

// validateConfig rejects unknown mapping names before preferences are used or updated.
func validateConfig(c Config) error {
	if c.Type != "mac" && c.Type != "win" {
		return fmt.Errorf("invalid keyboard type %q", c.Type)
	}
	for id, t := range c.KeyboardTypes {
		if t != "mac" && t != "win" {
			return fmt.Errorf("invalid keyboard override %s: %s", id, t)
		}
	}
	return nil
}

// saveConfig validates and submits one complete configuration to native storage.
// Success acknowledges the update, not completion of macOS's asynchronous disk write.
func saveConfig(preferences preferencesStore, c Config) error {
	if err := validateConfig(c); err != nil {
		return err
	}
	if c.KeyboardTypes == nil {
		c.KeyboardTypes = map[string]string{}
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return preferences.write(data)
}
