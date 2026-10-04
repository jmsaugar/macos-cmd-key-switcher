//go:build darwin && cgo

package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
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
	configPath := filepath.Join(home, "Library", "Application Support", "CmdKeySwitcher", "config.json")
	config, err := loadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		log.Fatal(err)
	}
	lockFile, err := os.OpenFile(configPath+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer lockFile.Close()
	if err = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		log.Fatal("another instance is running: ", err)
	}
	app := newApp(configPath, config)
	defer app.close()
	runNativeApp(app)
}
