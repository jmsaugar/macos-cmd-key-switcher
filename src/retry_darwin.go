//go:build darwin && cgo

package main

import (
	"log"
)

// Bounded automatic retry orchestration. Native one-shot timers call retryAutomatic.

func (a *App) resetRetries() {
	a.deps.cancelAutomaticRetry()
	a.retries = retryBudget{}
	a.retryMode = ""
}

func (a *App) queueRetry() {
	if seconds, ok := a.retries.next(); ok {
		a.deps.scheduleAutomaticRetry(seconds)
	} else {
		log.Print("Automatic a.retries exhausted; reconnect a keyboard, wake the Mac, or switch manually to try again")
	}
}

func (a *App) retryAutomatic() {
	if a.retryMode != "" {
		a.automaticType(a.retryMode)
	} else {
		a.detectDevices()
	}
}
