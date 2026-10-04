package main

import (
	"path/filepath"
	"testing"
)

func TestConfigRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	c, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	c.Type = "win"
	c.KeyboardTypes["1:2"] = "mac"
	if err = saveConfig(path, c); err != nil {
		t.Fatal(err)
	}
	got, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != "win" || got.KeyboardTypes["1:2"] != "mac" {
		t.Fatal("config did not round trip")
	}
}
