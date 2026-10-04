//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework IOKit -framework CoreFoundation
#include <stdlib.h>
#include "native.h"
*/
import "C"
import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"
)

var config Config
var configPath, lastDevices, lastError string
var lockFile *os.File
var retries retryBudget
var retryMode string
var forceRefresh bool
var observedDevices string
var executeMapping = applyMapping
var scheduleAutomaticRetry = func(seconds int) { C.scheduleRetry(C.double(seconds)) }
var cancelAutomaticRetry = func() { C.cancelRetry() }

func refreshMenu() {
	mode := C.CString(config.Type)
	defer C.free(unsafe.Pointer(mode))
	message := C.CString(lastError)
	defer C.free(unsafe.Pointer(message))
	C.updateMenu(mode, message)
}
func setType(t string) bool { return selectType(t, nil) }

func selectType(t string, remembered []Keyboard) bool {
	if err := executeMapping(config, t); err != nil {
		lastError = err.Error()
		log.Print(err)
		refreshMenu()
		return false
	}
	config.Type = t
	rememberKeyboards(&config, remembered, t)
	if err := saveConfig(configPath, config); err != nil {
		lastError = "Mapping applied, but config could not be saved: " + err.Error()
		log.Print(lastError)
	} else {
		lastError = ""
	}
	refreshMenu()
	return true
}

//export appTick
func appTick() {
	resetRetries()
	detectDevices()
}

var readKeyboards = connectedKeyboards

func connectedKeyboards() ([]Keyboard, error) {
	raw := C.keyboardJSON()
	if raw == nil {
		return nil, fmt.Errorf("could not enumerate keyboards")
	}
	defer C.free(unsafe.Pointer(raw))
	var devices []Keyboard
	if err := json.Unmarshal([]byte(C.GoString(raw)), &devices); err != nil {
		return nil, err
	}
	return devices, nil
}

func detectDevices() {
	devices, err := readKeyboards()
	if err != nil {
		lastError = err.Error()
		refreshMenu()
		queueRetry()
		return
	}
	updateDevices(devices)
}

func updateDevices(devices []Keyboard) {
	signature := fingerprint(devices)
	changed := signature != lastDevices
	observedDevices = signature
	if changed || forceRefresh {
		target := config.Type
		if changed {
			target = detectedType(devices, config)
		}
		forceRefresh = false
		automaticType(target)
		if retryMode == "" {
			lastDevices = signature
		}
	}
}

func resetRetries() {
	cancelAutomaticRetry()
	retries = retryBudget{}
	retryMode = ""
}

func queueRetry() {
	if seconds, ok := retries.next(); ok {
		scheduleAutomaticRetry(seconds)
	} else {
		log.Print("Automatic retries exhausted; reconnect a keyboard, wake the Mac, or switch manually to try again")
	}
}

func automaticType(mode string) {
	if !setType(mode) {
		retryMode = mode
		queueRetry()
	} else {
		retryMode = ""
		lastDevices = observedDevices
	}
}

//export appRetry
func appRetry() {
	if retryMode != "" {
		automaticType(retryMode)
	} else {
		detectDevices()
	}
}

//export appWake
func appWake() {
	forceRefresh = true
	appTick()
}

//export appSwitch
func appSwitch() {
	resetRetries()
	forceRefresh = false
	devices, err := readKeyboards()
	if err != nil {
		lastError = err.Error()
		log.Print(err)
		refreshMenu()
		return
	}
	observedDevices = fingerprint(devices)
	lastDevices = observedDevices
	t := "win"
	if config.Type == "win" {
		t = "mac"
	}
	selectType(t, devices)
}
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
	configPath = filepath.Join(home, "Library", "Application Support", "CmdKeySwitcher", "config.json")
	config, err = loadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		log.Fatal(err)
	}
	lockFile, err = os.OpenFile(configPath+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		log.Fatal(err)
	}
	if err = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		log.Fatal("another instance is running: ", err)
	}
	C.runApp()
}
