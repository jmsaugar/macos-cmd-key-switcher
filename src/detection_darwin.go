//go:build darwin && cgo

package main

// Connected-device state and automatic selection orchestration.
var lastDevices, observedDevices string
var forceRefresh bool
var readKeyboards = connectedKeyboards

func keyboardChanged() {
	refreshMenu()
	resetRetries()
	detectDevices()
}

func keyboardWake() {
	forceRefresh = true
	keyboardChanged()
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
		target := requestedType()
		if changed {
			target = detectedType(devices, config)
		}
		forceRefresh = false
		automaticType(target)
	}
}
