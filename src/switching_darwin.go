//go:build darwin && cgo

package main

import (
	"log"
)

// Application selection and errors; accessed only on the main OS thread.
var config Config
var lastError string
var executeMapping = applyMapping

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

func automaticType(mode string) {
	if !setType(mode) {
		retryMode = mode
		queueRetry()
	} else {
		retryMode = ""
		lastDevices = observedDevices
	}
}

func switchKeyboard() {
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
