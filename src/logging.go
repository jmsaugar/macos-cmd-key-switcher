package main

import (
	"os"
	"path/filepath"
)

// openAppLog opens app.log for append, creating directory if needed.
// This keeps diagnostics from Finder and login launches, which have no terminal.
// The caller must close the returned file.
func openAppLog(directory string) (*os.File, error) {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(directory, "app.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
}
