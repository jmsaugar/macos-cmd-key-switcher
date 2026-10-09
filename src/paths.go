package main

import "path/filepath"

// appDataDirectory returns the directory containing the app's single-instance lock.
func appDataDirectory(home string) string {
	return filepath.Join(home, "Library", "Application Support", "CmdKeySwitcher")
}

// instanceLockPath returns the lock file used to exclude another running instance.
func instanceLockPath(home string) string {
	return filepath.Join(appDataDirectory(home), "instance.lock")
}

// logsDirectory returns the app's log directory under home/Library/Logs.
func logsDirectory(home string) string {
	return filepath.Join(home, "Library", "Logs", "CmdKeySwitcher")
}
