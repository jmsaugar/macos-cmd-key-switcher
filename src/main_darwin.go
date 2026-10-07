//go:build darwin && cgo

package main

import (
	"flag"
	"io"
	"log"
	"os"
	"runtime"
)

// main loads preferences, locks the instance, opens logging, and runs the menu bar application.
//
// Command-line arguments are validated through flag parsing.
//
// Startup failures terminate the process.
func main() {
	runtime.LockOSThread()
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	// No optional runtime modes. Flag parsing rejects obsolete options before
	// the app can execute any mapping commands.
	flag.Parse()
	if flag.NArg() != 0 {
		log.Fatal("unexpected command-line arguments")
	}
	configPath := configFilePath(home)
	config, err := loadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}
	lock, err := acquireInstanceLock(configPath + ".lock")
	if err != nil {
		log.Fatal("Could not acquire the instance lock (another instance may be running): ", err)
	}
	defer lock.close()
	logFile, err := openAppLog(logsDirectory(home))
	if err != nil {
		log.Fatal(err)
	}
	// Callback closes the current log file when main returns.
	defer func() { logFile.Close() }()
	log.SetOutput(io.MultiWriter(os.Stderr, logFile))
	app := newApp(configPath, config)
	app.logsPath = logsDirectory(home)
	// Callback restores the instance lock and log destination after partial cleanup.
	// It returns nil on success, or a lock restoration or log opening error.
	app.deps.recoverAfterCleanup = func() error {
		if err := lock.restore(); err != nil {
			return err
		}
		replacement, err := openAppLog(app.logsPath)
		if err != nil {
			return err
		}
		log.SetOutput(io.MultiWriter(os.Stderr, replacement))
		logFile.Close()
		logFile = replacement
		return nil
	}
	defer app.close()
	runNativeApp(app)
}
