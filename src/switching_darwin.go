//go:build darwin && cgo

package main

import (
	"log"
)

// Application selection and errors; accessed only on the main OS thread.

// automaticType requests an automatic mapping with retries enabled on failure.
//
// The parameter mode is the target mapping; a owns the request queue.
func (a *App) automaticType(mode string) {
	a.requestMapping(mode, nil, true)
}

// switchKeyboard toggles the latest requested mode and captures connected-device preferences.
//
// The receiver a supplies current selection and device dependencies.
//
// Enumeration errors are logged and displayed in the menu.
func (a *App) switchKeyboard() {
	if a.stopping {
		return
	}
	a.resetRetries()
	a.forceRefresh = false
	devices, err := a.deps.readKeyboards()
	if err != nil {
		a.lastError = err.Error()
		log.Print(err)
		a.refreshMenu()
		return
	}
	a.observedDevices = fingerprint(devices)
	a.lastDevices = a.observedDevices
	t := "win"
	if a.requestedType() == "win" {
		t = "mac"
	}
	a.requestMapping(t, devices, false)
}
