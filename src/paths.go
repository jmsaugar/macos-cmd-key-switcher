package main

import "path/filepath"

// configFilePath returns the app's config.json path under home/Library/Application Support.
func configFilePath(home string) string {
	return filepath.Join(home, "Library", "Application Support", "CmdKeySwitcher", "config.json")
}

// logsDirectory returns the app's log directory under home/Library/Logs.
func logsDirectory(home string) string {
	return filepath.Join(home, "Library", "Logs", "CmdKeySwitcher")
}
