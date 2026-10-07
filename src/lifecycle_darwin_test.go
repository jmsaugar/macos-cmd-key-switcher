//go:build darwin && cgo

package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestCleanupWaitsForMappingAndDoesNotRecreateConfig verifies that cleanup waits for active
// mappings and does not recreate deleted configuration.
//
// The parameter t runs the test and reports assertion failures.
//
// Failures are reported through t.
func TestCleanupWaitsForMappingAndDoesNotRecreateConfig(t *testing.T) {
	a := testApp(t)
	home := t.TempDir()
	a.configPath, a.logsPath = configFilePath(home), logsDirectory(home)
	if err := saveConfig(a.configPath, a.config); err != nil {
		t.Fatal(err)
	}
	logFile, err := openAppLog(a.logsPath)
	if err != nil {
		t.Fatal(err)
	}
	defer logFile.Close()
	var order []string
	// Callback simulates removing launch-at-login registration.
	// It returns nil to simulate success.
	a.deps.unregisterStartup = func() error { order = append(order, "unregister"); return nil }
	// Callback simulates user-data cleanup.
	// It returns the error from deleting application data, or nil.
	a.deps.removeUserData = func() error { order = append(order, "remove"); return a.removeUserData() }
	// Callback handles application exit for this test.
	a.deps.quit = func() { order = append(order, "quit") }
	var running mappingRequest
	// Callback captures or completes mapping requests for this test.
	//
	// The parameter request is the mapping request to capture or complete.
	a.deps.startMapping = func(request mappingRequest) { running = request }
	a.requestMapping("win", nil, true)
	a.requestMapping("mac", nil, false) // pending work must not run during shutdown
	a.prepareForUninstall()
	if !reflect.DeepEqual(order, []string{"unregister"}) || a.pendingMapping != nil {
		t.Fatalf("cleanup did not wait for the active command: %v", order)
	}
	// Callback handles configuration persistence for this test and reports unexpected invocation
	// through t.
	//
	// The config path and configuration (unused by this stub).
	// It returns nil to simulate success.
	a.deps.saveConfig = func(string, Config) error { t.Fatal("saved during cleanup"); return nil }
	a.keyboardChanged()
	a.switchKeyboard()
	a.retryAutomatic()
	a.requestMapping("mac", nil, false)
	a.finishMapping(mappingResult{running, nil})
	if !reflect.DeepEqual(order, []string{"unregister", "remove", "quit"}) {
		t.Fatalf("wrong cleanup order: %v", order)
	}
	for _, path := range []string{filepath.Dir(a.configPath), a.logsPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("data directory remains: %s (%v)", path, err)
		}
	}
	config, err := loadConfig(a.configPath)
	if err != nil || config.Type != "mac" || len(config.KeyboardTypes) != 0 {
		t.Fatalf("reopening did not load defaults: %+v, %v", config, err)
	}
}

// TestClosePreservesDataAndStartupAndDrainsActiveSelection verifies that normal shutdown preserves
// data and startup registration while finishing active work.
//
// The parameter t runs the test and reports assertion failures.
//
// Failures are reported through t.
func TestClosePreservesDataAndStartupAndDrainsActiveSelection(t *testing.T) {
	a := testApp(t)
	// Callback simulates removing launch-at-login registration and reports unexpected invocation
	// through t.
	// It returns nil to simulate success.
	a.deps.unregisterStartup = func() error { t.Fatal("close unregistered startup"); return nil }
	// Callback simulates user-data cleanup and reports unexpected invocation through t.
	// It returns nil to simulate success.
	a.deps.removeUserData = func() error { t.Fatal("close deleted data"); return nil }
	quit := false
	// Callback handles application exit for this test.
	a.deps.quit = func() { quit = true }
	var running mappingRequest
	// Callback captures or completes mapping requests for this test.
	//
	// The parameter request is the mapping request to capture or complete.
	a.deps.startMapping = func(request mappingRequest) { running = request }
	keyboard := Keyboard{Vendor: 1234, ProductID: 5678, Serial: "unit-A"}
	a.requestMapping("win", []Keyboard{keyboard}, false)
	a.closeApp()
	if quit {
		t.Fatal("closed before active command finished")
	}
	a.finishMapping(mappingResult{running, nil})
	config, err := loadConfig(a.configPath)
	if !quit || err != nil || config.KeyboardTypes[unitKey(keyboard)] != "win" {
		t.Fatalf("close lost the active selection: %+v, %v, quit=%v", config, err, quit)
	}
}

// TestCleanupUnregisterFailurePreservesRunningAppAndData verifies that a startup unregister
// failure leaves the application running and preserves data.
//
// The parameter t runs the test and reports assertion failures.
//
// Failures are reported through t.
func TestCleanupUnregisterFailurePreservesRunningAppAndData(t *testing.T) {
	a := testApp(t)
	// Callback simulates removing launch-at-login registration.
	// It returns the injected error used to exercise failure handling.
	a.deps.unregisterStartup = func() error { return errors.New("denied") }
	// Callback simulates user-data cleanup and reports unexpected invocation through t.
	// It returns nil to simulate success.
	a.deps.removeUserData = func() error { t.Fatal("removed after unregister failure"); return nil }
	// Callback handles application exit for this test and reports unexpected invocation through t.
	a.deps.quit = func() { t.Fatal("quit after unregister failure") }
	var message string
	// Callback handles reported application errors for this test.
	//
	// The error message.
	a.deps.showError = func(value string) { message = value }
	a.prepareForUninstall()
	if a.stopping || message == "" {
		t.Fatal("unregister failure did not leave the app available")
	}
}

// TestCleanupDeletionFailureAllowsRetry verifies that a cleanup deletion failure restores
// operation and permits another cleanup attempt.
//
// The parameter t runs the test and reports assertion failures.
//
// Failures are reported through t.
func TestCleanupDeletionFailureAllowsRetry(t *testing.T) {
	a := testApp(t)
	// Callback simulates user-data cleanup.
	// It returns the injected error used to exercise failure handling.
	a.deps.removeUserData = func() error { return errors.New("permission denied") }
	// Callback handles reported application errors for this test.
	//
	// The error message.
	a.deps.showError = func(string) {}
	resumed, enabled, quit := false, false, false
	// Callback handles device watching recovery for this test.
	a.deps.resumeWatching = func() { resumed = true }
	// Callback handles menu availability changes for this test.
	//
	// The requested enabled state.
	a.deps.setMenuEnabled = func(value bool) { enabled = value }
	// Callback handles application exit for this test.
	a.deps.quit = func() { quit = true }
	a.prepareForUninstall()
	if a.stopping || !resumed || !enabled || quit || a.lastError == "" {
		t.Fatal("failed cleanup did not restore the UI")
	}
	// Callback simulates user-data cleanup.
	// It returns nil to simulate success.
	a.deps.removeUserData = func() error { return nil }
	a.prepareForUninstall()
	if !quit {
		t.Fatal("cleanup retry did not exit")
	}
}

// TestCleanupRecoveryFailureClosesInsteadOfResuming verifies that failed resource recovery after
// cleanup closes the application.
//
// The parameter t runs the test and reports assertion failures.
//
// Failures are reported through t.
func TestCleanupRecoveryFailureClosesInsteadOfResuming(t *testing.T) {
	a := testApp(t)
	// Callback simulates user-data cleanup.
	// It returns the injected error used to exercise failure handling.
	a.deps.removeUserData = func() error { return errors.New("partial deletion") }
	// Callback simulates startup-resource recovery.
	// It returns the injected error used to exercise failure handling.
	a.deps.recoverAfterCleanup = func() error { return errors.New("lock unavailable") }
	// Callback handles reported application errors for this test.
	//
	// The error message.
	a.deps.showError = func(string) {}
	// Callback handles device watching recovery for this test and reports unexpected invocation
	// through t.
	a.deps.resumeWatching = func() { t.Fatal("resumed without a valid instance lock") }
	quit := false
	// Callback handles application exit for this test.
	a.deps.quit = func() { quit = true }
	a.prepareForUninstall()
	if !quit || !a.stopping {
		t.Fatal("could not recover resources but did not close")
	}
}
