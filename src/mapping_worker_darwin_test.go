//go:build darwin && cgo

package main

import (
	"errors"
	"testing"
	"time"
)

func TestMappingWorkerDoesNotBlockOrUpdateState(t *testing.T) {
	isolateEvents(t)
	oldSignal := signalMappingComplete
	t.Cleanup(func() { signalMappingComplete = oldSignal })
	started, release, completed := make(chan struct{}), make(chan struct{}), make(chan struct{})
	executeMapping = func(Config, string) error { close(started); <-release; return nil }
	signalMappingComplete = func() { close(completed) }
	startMapping = launchMapping
	requestMapping("win", nil, true)
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not start")
	}
	if config.Type != "mac" {
		t.Fatal("state changed before completion")
	}
	close(release)
	select {
	case <-completed:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not complete")
	}
	if config.Type != "mac" {
		t.Fatal("worker changed main-thread state")
	}
	finishMapping(<-mappingResults)
	if config.Type != "win" {
		t.Fatal("main-thread completion did not update state")
	}
}

func TestPendingRequestsAreSerializedAndLatestWins(t *testing.T) {
	isolateEvents(t)
	var launched []mappingRequest
	startMapping = func(request mappingRequest) { launched = append(launched, request) }
	devices := []Keyboard{{ID: "external", Vendor: 1234}}
	readKeyboards = func() ([]Keyboard, error) { return devices, nil }
	updateDevices(devices) // running automatic win
	appSwitch()            // pending manual mac
	appSwitch()            // pending manual win replaces mac
	if len(launched) != 1 || pendingMapping.mode != "win" {
		t.Fatal("overlapping command or wrong requested toggle")
	}
	finishMapping(mappingResult{launched[0], errors.New("superseded failure")})
	if len(launched) != 2 || launched[1].mode != "win" || launched[1].automatic {
		t.Fatal("latest manual request not launched")
	}
	if retries.attempts != 0 {
		t.Fatal("superseded failure scheduled a retry")
	}
	finishMapping(mappingResult{launched[1], nil})
	if config.Type != "win" || config.KeyboardTypes[modelKey(devices[0])] != "win" {
		t.Fatal("latest selection not remembered")
	}
	if activeMapping != nil || pendingMapping != nil {
		t.Fatal("queue did not drain")
	}
}
