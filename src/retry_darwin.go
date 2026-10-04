//go:build darwin && cgo

package main

import (
	"log"
)

// Bounded automatic retry orchestration. Native one-shot timers call retryAutomatic.
var retries retryBudget
var retryMode string
var scheduleAutomaticRetry = scheduleNativeRetry
var cancelAutomaticRetry = cancelNativeRetry

func resetRetries() {
	cancelAutomaticRetry()
	retries = retryBudget{}
	retryMode = ""
}

func queueRetry() {
	if seconds, ok := retries.next(); ok {
		scheduleAutomaticRetry(seconds)
	} else {
		log.Print("Automatic retries exhausted; reconnect a keyboard, wake the Mac, or switch manually to try again")
	}
}

func retryAutomatic() {
	if retryMode != "" {
		automaticType(retryMode)
	} else {
		detectDevices()
	}
}
