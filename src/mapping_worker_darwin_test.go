//go:build darwin && cgo

package main

import (
	"errors"
	"testing"
	"time"
)

// TestMappingWorkerDoesNotBlockOrUpdateState verifies that mapping workers run asynchronously
// without changing main-thread application state.
//
// The parameter t runs the test and reports assertion failures.
//
// Failures are reported through t.
func TestMappingWorkerDoesNotBlockOrUpdateState(t *testing.T) {
	a := testApp(t)
	started, release, completed := make(chan struct{}), make(chan struct{}), make(chan struct{})
	// Callback simulates mapping execution for this test.
	//
	// The requested mapping mode (unused by this stub).
	// It returns nil to simulate success.
	a.deps.executeMapping = func(string) error { close(started); <-release; return nil }
	// Callback signals worker completion to the test.
	a.deps.signalMappingComplete = func() { close(completed) }
	a.deps.startMapping = a.launchMapping
	a.requestMapping("win", nil, true)
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not start")
	}
	if a.config.Type != "mac" {
		t.Fatal("state changed before completion")
	}
	close(release)
	select {
	case <-completed:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not complete")
	}
	if a.config.Type != "mac" {
		t.Fatal("worker changed main-thread state")
	}
	a.finishMapping(<-a.mappingResults)
	if a.config.Type != "win" {
		t.Fatal("main-thread completion did not update state")
	}
}

// TestPendingRequestsAreSerializedAndLatestWins verifies that mapping requests execute serially
// and the latest pending request wins.
//
// The parameter t runs the test and reports assertion failures.
//
// Failures are reported through t.
func TestPendingRequestsAreSerializedAndLatestWins(t *testing.T) {
	a := testApp(t)
	var launched []mappingRequest
	// Callback captures or completes mapping requests for this test.
	//
	// The parameter request is the mapping request to capture or complete.
	a.deps.startMapping = func(request mappingRequest) { launched = append(launched, request) }
	devices := []Keyboard{{ID: "external", Vendor: 1234}}
	// Callback supplies the keyboard snapshot for this test.
	// It returns the test keyboard snapshot and nil.
	a.deps.readKeyboards = func() ([]Keyboard, error) { return devices, nil }
	a.updateDevices(devices) // running automatic win
	a.switchKeyboard()       // pending manual mac
	a.switchKeyboard()       // pending manual win replaces mac
	if len(launched) != 1 || a.pendingMapping.mode != "win" {
		t.Fatal("overlapping command or wrong requested toggle")
	}
	a.finishMapping(mappingResult{launched[0], errors.New("superseded failure")})
	if len(launched) != 2 || launched[1].mode != "win" || launched[1].automatic {
		t.Fatal("latest manual request not launched")
	}
	if a.retries.attempts != 0 {
		t.Fatal("superseded failure scheduled a retry")
	}
	a.finishMapping(mappingResult{launched[1], nil})
	if a.config.Type != "win" || a.config.KeyboardTypes[modelKey(devices[0])] != "win" {
		t.Fatal("latest selection not remembered")
	}
	if a.activeMapping != nil || a.pendingMapping != nil {
		t.Fatal("queue did not drain")
	}
}
