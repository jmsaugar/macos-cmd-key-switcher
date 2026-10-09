//go:build darwin && cgo

// Cmd Key Switcher selects a global keyboard modifier mapping from connected
// keyboards and saved device preferences. Its menu supports manual switching,
// optional launch at login, and cleanup before uninstalling.
package main

import (
	"flag"
	"io"
	"log"
	"os"
	"runtime"
)

// main acquires the instance lock, loads preferences, and runs the native UI.
// It locks the main goroutine to its OS thread before calling Cocoa.
// Startup errors terminate the process.
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
	lock, err := acquireInstanceLock(instanceLockPath(home))
	if err != nil {
		log.Fatal("Could not acquire the instance lock (another instance may be running): ", err)
	}
	defer lock.close()
	preferences := nativePreferences{}
	config, err := loadConfig(preferences)
	if err != nil {
		log.Fatal(err)
	}
	logFile, err := openAppLog(logsDirectory(home))
	if err != nil {
		log.Fatal(err)
	}
	// Cleanup recovery can replace logFile; close the current file on return.
	defer func() { logFile.Close() }()
	log.SetOutput(io.MultiWriter(os.Stderr, logFile))
	app := newApp(config, preferences)
	app.dataPath = appDataDirectory(home)
	app.logsPath = logsDirectory(home)
	// Partial cleanup can unlink files while their old descriptors remain open.
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
