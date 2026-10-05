//go:build darwin && cgo

package main

import (
	"flag"
	"io"
	"log"
	"os"
	"runtime"
)

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
	defer func() { logFile.Close() }()
	log.SetOutput(io.MultiWriter(os.Stderr, logFile))
	app := newApp(configPath, config)
	app.logsPath = logsDirectory(home)
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
