//go:build darwin && cgo

package main

import (
	"log"
)

// automaticType queues an automatic mapping; an unsuperseded command failure
// can schedule a bounded retry.
func (a *App) automaticType(mode string) {
	a.requestMapping(mode, nil, true)
}

// switchKeyboard toggles the latest requested mode using a fresh device snapshot.
// External-device preferences are saved only after a successful, unsuperseded
// command. Enumeration failure leaves the selection unchanged.
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
