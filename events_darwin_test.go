//go:build darwin && cgo

package main

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func isolateEvents(t *testing.T) {
	t.Helper()
	oldConfig, oldDry, oldLast, oldObserved, oldError := config, dryRun, lastDevices, observedDevices, lastError
	oldRetries, oldMode, oldForce := retries, retryMode, forceRefresh
	oldRead := readKeyboards
	oldExec, oldSchedule, oldCancel := executeMapping, scheduleAutomaticRetry, cancelAutomaticRetry
	t.Cleanup(func() {
		readKeyboards = oldRead
		config, dryRun, lastDevices, observedDevices, lastError = oldConfig, oldDry, oldLast, oldObserved, oldError
		retries, retryMode, forceRefresh = oldRetries, oldMode, oldForce
		executeMapping, scheduleAutomaticRetry, cancelAutomaticRetry = oldExec, oldSchedule, oldCancel
	})
	readKeyboards = func() ([]Keyboard, error) { return []Keyboard{{ID: "external", Vendor: 1234}}, nil }
	config = defaults()
	dryRun = true
	lastDevices = ""
	observedDevices = ""
	retries = retryBudget{}
	retryMode = ""
	forceRefresh = false
	cancelAutomaticRetry = func() {}
	scheduleAutomaticRetry = func(int) { t.Fatal("unexpected retry") }
	executeMapping = func(Config, string) error { t.Fatal("unexpected HID command"); return nil }
}

func TestDeviceEventsPreserveManualSelection(t *testing.T) {
	isolateEvents(t)
	windows := []Keyboard{{ID: "external", Vendor: 1234}}
	updateDevices(windows)
	if config.Type != "win" {
		t.Fatal("connection did not select win")
	}
	appSwitch()
	updateDevices(windows)
	if config.Type != "mac" {
		t.Fatal("duplicate notification replaced manual selection")
	}
	forceRefresh = true
	updateDevices(windows)
	if config.Type != "mac" {
		t.Fatal("wake replaced manual selection")
	}
	updateDevices(nil)
	if config.Type != "mac" {
		t.Fatal("disconnect did not select mac")
	}
}

func TestAutomaticFailureRetriesAndManualCancellation(t *testing.T) {
	isolateEvents(t)
	dryRun = false
	var delays []int
	scheduleAutomaticRetry = func(seconds int) { delays = append(delays, seconds) }
	calls := 0
	executeMapping = func(Config, string) error { calls++; return errors.New("simulated failure") }
	updateDevices([]Keyboard{{ID: "external", Vendor: 1234}})
	for i := 0; i < 3; i++ {
		appRetry()
	}
	if calls != 4 || !reflect.DeepEqual(delays, []int{2, 4, 8}) {
		t.Fatalf("calls=%d delays=%v", calls, delays)
	}
	if config.Type != "mac" {
		t.Fatal("failure changed state")
	}
	dryRun = true
	cancelled := false
	cancelAutomaticRetry = func() { cancelled = true }
	appSwitch()
	if !cancelled || retryMode != "" || retries.attempts != 0 {
		t.Fatal("manual switch did not cancel retries")
	}
	appSwitch() // choose mac manually for the still-connected Windows keyboard
	updateDevices([]Keyboard{{ID: "external", Vendor: 1234}})
	if config.Type != "mac" {
		t.Fatal("duplicate event retried superseded automatic selection")
	}
}

func TestManualSwitchPersistsConnectedUnit(t *testing.T) {
	isolateEvents(t)
	oldPath := configPath
	t.Cleanup(func() { configPath = oldPath })
	configPath = filepath.Join(t.TempDir(), "config.json")
	keyboard := Keyboard{ID: "new", Vendor: 1234, ProductID: 5678, Serial: "unit-A"}
	readKeyboards = func() ([]Keyboard, error) { return []Keyboard{keyboard}, nil }
	config.Type = "win"
	dryRun = false
	executeMapping = func(Config, string) error { return nil }
	appSwitch()
	saved, err := loadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Type != "mac" || saved.KeyboardTypes[unitKey(keyboard)] != "mac" {
		t.Fatal("manual unit preference not saved")
	}
	executeMapping = func(Config, string) error { return errors.New("simulated failure") }
	appSwitch()
	if config.KeyboardTypes[unitKey(keyboard)] != "mac" {
		t.Fatal("failed command changed preference")
	}
}
