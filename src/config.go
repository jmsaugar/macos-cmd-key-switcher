package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Type string `json:"type"`
	// Keys are vendor:product or vendor:product:serial:<escaped serial>.
	KeyboardTypes map[string]string `json:"keyboard_types"`
}

func defaults() Config {
	return Config{Type: "mac", KeyboardTypes: map[string]string{}}
}
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
