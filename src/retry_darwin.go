//go:build darwin && cgo

package main

import (
	"log"
)

// Bounded automatic retry orchestration. Native one-shot timers call retryAutomatic.

// resetRetries cancels any scheduled retry and resets the automatic retry budget and mode.
//
// The receiver a owns the retry state.
func (a *App) resetRetries() {
	a.deps.cancelAutomaticRetry()
	a.retries = retryBudget{}
	a.retryMode = ""
}

// queueRetry schedules the next automatic retry if the budget permits and shutdown has not
// started.
//
// The receiver a supplies the retry budget and timer dependency.
//
// Budget exhaustion is logged.
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

// retryAutomatic retries a failed automatic mapping or device detection unless stopping.
//
// The receiver a supplies the retry mode and device state.
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
