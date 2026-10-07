//go:build darwin && cgo

package main

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

// testApp creates an isolated application with synchronous mapping and stubbed native
// dependencies.
//
// The parameter t supplies temporary storage and reports unexpected callbacks.
// It returns the application configured for tests.
func testApp(t *testing.T) *App {
	t.Helper()
	a := newApp(filepath.Join(t.TempDir(), "config.json"), defaults())
	// Callback supplies the keyboard snapshot for this test.
	// It returns the test keyboard snapshot and nil.
	a.deps.readKeyboards = func() ([]Keyboard, error) { return []Keyboard{{ID: "external", Vendor: 1234}}, nil }
	// Callback handles retry cancellation for this test.
	a.deps.cancelAutomaticRetry = func() {}
	// Callback handles retry scheduling for this test and reports unexpected invocation through t.
	//
	// The retry delay in seconds.
	a.deps.scheduleAutomaticRetry = func(int) { t.Fatal("unexpected retry") }
	// Callback simulates mapping execution for this test.
	//
	// The requested mapping mode (unused by this stub).
	// It returns nil to simulate success.
	a.deps.executeMapping = func(string) error { return nil }
	// Callback accepts menu updates without invoking Cocoa.
	//
	// The applied mode and error message (unused by this stub).
	a.deps.updateMenu = func(string, string) {}
	// Callback simulates removing launch-at-login registration.
	// It returns nil to simulate success.
	a.deps.unregisterStartup = func() error { return nil }
	// Callback simulates user-data cleanup.
	// It returns nil to simulate success.
	a.deps.removeUserData = func() error { return nil }
	// Callback accepts device watching shutdown without invoking Cocoa.
	a.deps.stopWatching = func() {}
	// Callback handles device watching recovery for this test.
	a.deps.resumeWatching = func() {}
	// Callback handles menu availability changes for this test.
	//
	// The requested enabled state.
	a.deps.setMenuEnabled = func(bool) {}
	// Callback handles application exit for this test.
	a.deps.quit = func() {}
	// Callback handles reported application errors for this test and reports unexpected invocation
	// through t.
	//
	// The error message.
	a.deps.showError = func(message string) { t.Fatal(message) }
	// Callback captures or completes mapping requests for this test.
	//
	// The parameter request is the mapping request to capture or complete.
	a.deps.startMapping = func(request mappingRequest) {
		a.finishMapping(mappingResult{request, a.deps.executeMapping(request.mode)})
	}
	return a
}

// TestDeviceEventsPreserveManualSelection verifies that duplicate device events and wake preserve
// a manual selection.
//
// The parameter t runs the test and reports assertion failures.
//
// Failures are reported through t.
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
//
// The parameter t runs the test and reports assertion failures.
//
// Failures are reported through t.
func TestAutomaticFailureRetriesAndManualCancellation(t *testing.T) {
	a := testApp(t)
	var delays []int
	// Callback records each requested retry delay.
	//
	// The retry delay in seconds.
	a.deps.scheduleAutomaticRetry = func(seconds int) { delays = append(delays, seconds) }
	calls := 0
	// Callback simulates mapping execution for this test.
	//
	// The requested mapping mode (unused by this stub).
	// It returns the injected error used to exercise failure handling.
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
	// Callback simulates mapping execution for this test.
	//
	// The requested mapping mode (unused by this stub).
	// It returns nil to simulate success.
	a.deps.executeMapping = func(string) error { return nil }
	cancelled := false
	// Callback records retry cancellation.
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
//
// The parameter t runs the test and reports assertion failures.
//
// Failures are reported through t.
func TestManualSwitchPersistsConnectedUnit(t *testing.T) {
	a := testApp(t)
	a.configPath = filepath.Join(t.TempDir(), "config.json")
	keyboard := Keyboard{ID: "new", Vendor: 1234, ProductID: 5678, Serial: "unit-A"}
	// Callback supplies the keyboard snapshot for this test.
	// It returns the test keyboard snapshot and nil.
	a.deps.readKeyboards = func() ([]Keyboard, error) { return []Keyboard{keyboard}, nil }
	a.config.Type = "win"
	// Callback simulates mapping execution for this test.
	//
	// The requested mapping mode (unused by this stub).
	// It returns nil to simulate success.
	a.deps.executeMapping = func(string) error { return nil }
	a.switchKeyboard()
	saved, err := loadConfig(a.configPath)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Type != "mac" || saved.KeyboardTypes[unitKey(keyboard)] != "mac" {
		t.Fatal("manual unit preference not saved")
	}
	// Callback simulates mapping execution for this test.
	//
	// The requested mapping mode (unused by this stub).
	// It returns the injected error used to exercise failure handling.
	a.deps.executeMapping = func(string) error { return errors.New("simulated failure") }
	a.switchKeyboard()
	if a.config.KeyboardTypes[unitKey(keyboard)] != "mac" {
		t.Fatal("failed command changed preference")
	}
}
