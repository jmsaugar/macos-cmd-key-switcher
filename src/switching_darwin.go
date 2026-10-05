//go:build darwin && cgo

package main

import (
	"log"
)

// Application selection and errors; accessed only on the main OS thread.

func (a *App) automaticType(mode string) {
	a.requestMapping(mode, nil, true)
}

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
