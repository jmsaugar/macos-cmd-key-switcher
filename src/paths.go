package main

import "path/filepath"

func configFilePath(home string) string {
	return filepath.Join(home, "Library", "Application Support", "CmdKeySwitcher", "config.json")
}

func logsDirectory(home string) string {
	return filepath.Join(home, "Library", "Logs", "CmdKeySwitcher")
}
