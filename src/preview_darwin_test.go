//go:build darwin && cgo

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreviewSwitchDoesNotWriteConfig(t *testing.T) {
	oldConfig, oldPath, oldDry, oldError := config, configPath, dryRun, lastError
	defer func() { config, configPath, dryRun, lastError = oldConfig, oldPath, oldDry, oldError }()
	config = defaults()
	configPath = filepath.Join(t.TempDir(), "must-not-create", "config.json")
	dryRun = true
	for _, mode := range []string{"win", "mac"} {
		if !setType(mode) || config.Type != mode {
			t.Fatalf("preview did not switch to %s", mode)
		}
	}
	if _, err := os.Stat(filepath.Dir(configPath)); !os.IsNotExist(err) {
		t.Fatalf("preview created config directory: %v", err)
	}
}
