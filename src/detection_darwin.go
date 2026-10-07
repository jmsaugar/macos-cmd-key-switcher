//go:build darwin && cgo

package main

// keyboardChanged resets the retry budget and enumerates devices unless stopping.
func (a *App) keyboardChanged() {
	if a.stopping {
		return
	}
	a.refreshMenu()
	a.resetRetries()
	a.detectDevices()
}

// keyboardWake requests reapplication of the latest desired mode after wake.
// If the device signature changed, detection chooses a new mode instead.
func (a *App) keyboardWake() {
	a.forceRefresh = true
	a.keyboardChanged()
}

// detectDevices enumerates keyboards and updates selection, scheduling a retry
// if enumeration fails.
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

// updateDevices selects a mode when the device signature changes.
// A forced refresh with the same signature reapplies the latest desired mode,
// preserving a manual selection.
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
