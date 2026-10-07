//go:build darwin && cgo

package main

import (
	"log"
)

// resetRetries cancels the retry timer and resets the retry budget and mode.
func (a *App) resetRetries() {
	a.deps.cancelAutomaticRetry()
	a.retries = retryBudget{}
	a.retryMode = ""
}

// queueRetry schedules the next retry unless stopping or the budget is exhausted.
func (a *App) queueRetry() {
	if a.stopping {
		return
	}
	if seconds, ok := a.retries.next(); ok {
		a.deps.scheduleAutomaticRetry(seconds)
	} else {
		log.Print("Automatic a.retries exhausted; reconnect a keyboard, wake the Mac, or switch manually to try again")
	}
}

// retryAutomatic retries the failed automatic mapping, or repeats device
// enumeration if no mapping retry is pending. It does nothing during shutdown.
func (a *App) retryAutomatic() {
	if a.stopping {
		return
	}
	if a.retryMode != "" {
		a.automaticType(a.retryMode)
	} else {
		a.detectDevices()
	}
}
