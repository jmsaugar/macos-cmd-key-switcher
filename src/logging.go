package main

import (
	"os"
	"path/filepath"
)

// Finder and login launches have no terminal, so keep diagnostics in an app log.
// openAppLog opens the persistent app log for append, creating its directory when needed.
//
// The parameter directory is the application log directory.
// It returns the open file and nil, or nil and a directory or open error; the caller closes the
// file.
func openAppLog(directory string) (*os.File, error) {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(directory, "app.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
}
