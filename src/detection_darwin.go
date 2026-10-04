//go:build darwin && cgo

package main

// Connected-device state and automatic selection orchestration.

func (a *App) keyboardChanged() {
	a.refreshMenu()
	a.resetRetries()
	a.detectDevices()
}

func (a *App) keyboardWake() {
	a.forceRefresh = true
	a.keyboardChanged()
}

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
