//go:build darwin && cgo

package main

// Connected-device state and automatic selection orchestration.

// keyboardChanged refreshes the menu and restarts device detection unless shutdown is underway.
//
// The receiver a is the application handling the event.
func (a *App) keyboardChanged() {
	if a.stopping {
		return
	}
	a.refreshMenu()
	a.resetRetries()
	a.detectDevices()
}

// keyboardWake forces mapping reapplication after wake and triggers device detection.
//
// The receiver a is the application handling wake.
func (a *App) keyboardWake() {
	a.forceRefresh = true
	a.keyboardChanged()
}

// detectDevices enumerates keyboards and updates selection, scheduling a retry on failure.
//
// The receiver a supplies the device reader and selection state.
func (a *App) detectDevices() {
	devices, err := a.deps.readKeyboards()
	if err != nil {
		a.lastError = err.Error()
		a.refreshMenu()
		a.queueRetry()
		return
	}
	a.updateDevices(devices)
}

// updateDevices tracks the device fingerprint and selects or reapplies a mapping when needed.
//
// The parameter devices is the latest snapshot; a holds preferences and prior observations.
func (a *App) updateDevices(devices []Keyboard) {
	signature := fingerprint(devices)
	changed := signature != a.lastDevices
	a.observedDevices = signature
	if changed || a.forceRefresh {
		target := a.requestedType()
		if changed {
			target = detectedType(devices, a.config)
		}
		a.forceRefresh = false
		a.automaticType(target)
	}
}
