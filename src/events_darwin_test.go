//go:build darwin && cgo

package main

import (
	"errors"
	"reflect"
	"testing"
)

// testApp creates isolated state with synchronous mapping completion and mocked
// native operations. Preferences stay in memory; HID commands, login
// registration, and Cocoa UI are never invoked.
func testApp(t *testing.T) *App {
	t.Helper()
	a := newApp(defaults(), &memoryPreferences{})
	a.deps.readKeyboards = func() ([]Keyboard, error) { return []Keyboard{{ID: "external", Vendor: 1234}}, nil }
	a.deps.cancelAutomaticRetry = func() {}
	a.deps.scheduleAutomaticRetry = func(int) { t.Fatal("unexpected retry") }
	a.deps.executeMapping = func(string) error { return nil }
	a.deps.updateMenu = func(string, string) {}
	a.deps.unregisterStartup = func() error { return nil }
	a.deps.removeUserData = func() error { return nil }
	a.deps.stopWatching = func() {}
	a.deps.resumeWatching = func() {}
	a.deps.setMenuEnabled = func(bool) {}
	a.deps.quit = func() {}
	a.deps.showError = func(message string) { t.Fatal(message) }
	a.deps.startMapping = func(request mappingRequest) {
		a.finishMapping(mappingResult{request, a.deps.executeMapping(request.mode)})
	}
	return a
}

// TestDeviceEventsPreserveManualSelection verifies that duplicate device events and wake preserve
// a manual selection.
func TestDeviceEventsPreserveManualSelection(t *testing.T) {
	a := testApp(t)
	windows := []Keyboard{{ID: "external", Vendor: 1234}}
	a.updateDevices(windows)
	if a.config.Type != "win" {
		t.Fatal("connection did not select win")
	}
	a.switchKeyboard()
	a.updateDevices(windows)
	if a.config.Type != "mac" {
		t.Fatal("duplicate notification replaced manual selection")
	}
	a.forceRefresh = true
	a.updateDevices(windows)
	if a.config.Type != "mac" {
		t.Fatal("wake replaced manual selection")
	}
	a.updateDevices(nil)
	if a.config.Type != "mac" {
		t.Fatal("disconnect did not select mac")
	}
}

// TestAutomaticFailureRetriesAndManualCancellation verifies that automatic failures retry within
// the budget and manual switching cancels retries.
func TestAutomaticFailureRetriesAndManualCancellation(t *testing.T) {
	a := testApp(t)
	var delays []int
	a.deps.scheduleAutomaticRetry = func(seconds int) { delays = append(delays, seconds) }
	calls := 0
	a.deps.executeMapping = func(string) error { calls++; return errors.New("simulated failure") }
	a.updateDevices([]Keyboard{{ID: "external", Vendor: 1234}})
	for i := 0; i < 3; i++ {
		a.retryAutomatic()
	}
	if calls != 4 || !reflect.DeepEqual(delays, []int{2, 4, 8}) {
		t.Fatalf("calls=%d delays=%v", calls, delays)
	}
	if a.config.Type != "mac" {
		t.Fatal("failure changed state")
	}
	a.deps.executeMapping = func(string) error { return nil }
	cancelled := false
	a.deps.cancelAutomaticRetry = func() { cancelled = true }
	a.switchKeyboard()
	if !cancelled || a.retryMode != "" || a.retries.attempts != 0 {
		t.Fatal("manual switch did not cancel retries")
	}
	a.switchKeyboard() // choose mac manually for the still-connected Windows keyboard
	a.updateDevices([]Keyboard{{ID: "external", Vendor: 1234}})
	if a.config.Type != "mac" {
		t.Fatal("duplicate event retried superseded automatic selection")
	}
}

// TestManualSwitchPersistsConnectedUnit verifies that successful manual switches persist unit
// preferences and failures preserve them.
func TestManualSwitchPersistsConnectedUnit(t *testing.T) {
	a := testApp(t)
	keyboard := Keyboard{ID: "new", Vendor: 1234, ProductID: 5678, Serial: "unit-A"}
	a.deps.readKeyboards = func() ([]Keyboard, error) { return []Keyboard{keyboard}, nil }
	a.config.Type = "win"
	a.deps.executeMapping = func(string) error { return nil }
	a.switchKeyboard()
	saved, err := loadConfig(a.preferences)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Type != "mac" || saved.KeyboardTypes[unitKey(keyboard)] != "mac" {
		t.Fatal("manual unit preference not saved")
	}
	a.deps.executeMapping = func(string) error { return errors.New("simulated failure") }
	a.switchKeyboard()
	if a.config.KeyboardTypes[unitKey(keyboard)] != "mac" {
		t.Fatal("failed command changed preference")
	}
}
