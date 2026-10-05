package main

import (
	"os"
	"path/filepath"
)

// Finder and login launches have no terminal, so keep diagnostics in an app log.
func openAppLog(directory string) (*os.File, error) {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(directory, "app.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
}
