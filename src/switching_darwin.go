//go:build darwin && cgo

package main

import (
	"log"
)

// Application selection and errors; accessed only on the main OS thread.
var config Config
var lastError string
var executeMapping = applyMapping

func automaticType(mode string) {
	requestMapping(mode, nil, true)
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
	if requestedType() == "win" {
		t = "mac"
	}
	requestMapping(t, devices, false)
}
