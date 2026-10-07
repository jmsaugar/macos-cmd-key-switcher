package main

import "path/filepath"

// configFilePath builds the per-user configuration file location.
//
// The parameter home is the user home directory.
// It returns the configuration file path under Library/Application Support.
func configFilePath(home string) string {
	return filepath.Join(home, "Library", "Application Support", "CmdKeySwitcher", "config.json")
}

// logsDirectory builds the per-user application log directory.
//
// The parameter home is the user home directory.
// It returns the log directory path under Library/Logs.
func logsDirectory(home string) string {
	return filepath.Join(home, "Library", "Logs", "CmdKeySwitcher")
}
