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
// mappings and does not recreate cleared preferences.
func TestCleanupWaitsForMappingAndDoesNotRecreateConfig(t *testing.T) {
	a := testApp(t)
	home := t.TempDir()
	a.dataPath, a.logsPath = appDataDirectory(home), logsDirectory(home)
	lock, err := acquireInstanceLock(instanceLockPath(home))
	if err != nil {
		t.Fatal(err)
	}
	defer lock.close()
	if err := saveConfig(a.preferences, a.config); err != nil {
		t.Fatal(err)
	}
	logFile, err := openAppLog(a.logsPath)
	if err != nil {
		t.Fatal(err)
	}
	defer logFile.Close()
	var order []string
	a.deps.unregisterStartup = func() error { order = append(order, "unregister"); return nil }
	a.deps.removeUserData = func() error { order = append(order, "remove"); return a.removeUserData() }
	a.deps.quit = func() { order = append(order, "quit") }
	var running mappingRequest
	a.deps.startMapping = func(request mappingRequest) { running = request }
	a.requestMapping("win", nil, true)
	a.requestMapping("mac", nil, false) // pending work must not run during shutdown
	a.prepareForUninstall()
	if !reflect.DeepEqual(order, []string{"unregister"}) || a.pendingMapping != nil {
		t.Fatalf("cleanup did not wait for the active command: %v", order)
	}
	a.deps.saveConfig = func(Config) error { t.Fatal("saved during cleanup"); return nil }
	a.keyboardChanged()
	a.switchKeyboard()
	a.retryAutomatic()
	a.requestMapping("mac", nil, false)
	a.finishMapping(mappingResult{running, nil})
	if !reflect.DeepEqual(order, []string{"unregister", "remove", "quit"}) {
		t.Fatalf("wrong cleanup order: %v", order)
	}
	for _, path := range []string{a.dataPath, a.logsPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("data directory remains: %s (%v)", path, err)
		}
	}
	config, err := loadConfig(a.preferences)
	if err != nil || config.Type != "mac" || len(config.KeyboardTypes) != 0 {
		t.Fatalf("reopening did not load defaults: %+v, %v", config, err)
	}
}

// TestClosePreservesDataAndStartupAndDrainsActiveSelection verifies that normal shutdown preserves
// data and startup registration while finishing active work.
func TestClosePreservesDataAndStartupAndDrainsActiveSelection(t *testing.T) {
	a := testApp(t)
	a.deps.unregisterStartup = func() error { t.Fatal("close unregistered startup"); return nil }
	a.deps.removeUserData = func() error { t.Fatal("close deleted data"); return nil }
	quit := false
	a.deps.quit = func() { quit = true }
	var running mappingRequest
	a.deps.startMapping = func(request mappingRequest) { running = request }
	keyboard := Keyboard{Vendor: 1234, ProductID: 5678, Serial: "unit-A"}
	a.requestMapping("win", []Keyboard{keyboard}, false)
	a.closeApp()
	if quit {
		t.Fatal("closed before active command finished")
	}
	a.finishMapping(mappingResult{running, nil})
	config, err := loadConfig(a.preferences)
	if !quit || err != nil || config.KeyboardTypes[unitKey(keyboard)] != "win" {
		t.Fatalf("close lost the active selection: %+v, %v, quit=%v", config, err, quit)
	}
}

// TestFileRemovalFailurePreservesPreferences verifies cleanup does not clear preferences
// until the logs and lock directory have been removed successfully.
func TestFileRemovalFailurePreservesPreferences(t *testing.T) {
	a := testApp(t)
	if err := saveConfig(a.preferences, Config{Type: "win"}); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(parent, []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	a.logsPath = filepath.Join(parent, "logs")
	if err := a.removeUserData(); err == nil {
		t.Fatal("expected file removal to fail")
	}
	config, err := loadConfig(a.preferences)
	if err != nil || config.Type != "win" {
		t.Fatalf("file removal failure cleared preferences: %+v, %v", config, err)
	}
}

// TestPreferenceClearFailureAllowsRetry verifies a bridge failure keeps cleanup retryable.
func TestPreferenceClearFailureAllowsRetry(t *testing.T) {
	a := testApp(t)
	preferences := a.preferences.(*memoryPreferences)
	preferences.clearErr = errors.New("preferences unavailable")
	a.deps.removeUserData = a.removeUserData
	a.deps.showError = func(string) {}
	quit := false
	a.deps.quit = func() { quit = true }
	a.prepareForUninstall()
	if quit || a.stopping || a.lastError == "" {
		t.Fatal("preference clear failure did not restore operation")
	}
	preferences.clearErr = nil
	a.prepareForUninstall()
	if !quit {
		t.Fatal("preference cleanup retry did not exit")
	}
}

// TestCleanupUnregisterFailurePreservesRunningAppAndData verifies that a startup unregister
// failure leaves the application running and preserves data.
func TestCleanupUnregisterFailurePreservesRunningAppAndData(t *testing.T) {
	a := testApp(t)
	a.deps.unregisterStartup = func() error { return errors.New("denied") }
	a.deps.removeUserData = func() error { t.Fatal("removed after unregister failure"); return nil }
	a.deps.quit = func() { t.Fatal("quit after unregister failure") }
	var message string
	a.deps.showError = func(value string) { message = value }
	a.prepareForUninstall()
	if a.stopping || message == "" {
		t.Fatal("unregister failure did not leave the app available")
	}
}

// TestCleanupDeletionFailureAllowsRetry verifies that a cleanup deletion failure restores
// operation and permits another cleanup attempt.
func TestCleanupDeletionFailureAllowsRetry(t *testing.T) {
	a := testApp(t)
	a.deps.removeUserData = func() error { return errors.New("permission denied") }
	a.deps.showError = func(string) {}
	resumed, enabled, quit := false, false, false
	a.deps.resumeWatching = func() { resumed = true }
	a.deps.setMenuEnabled = func(value bool) { enabled = value }
	a.deps.quit = func() { quit = true }
	a.prepareForUninstall()
	if a.stopping || !resumed || !enabled || quit || a.lastError == "" {
		t.Fatal("failed cleanup did not restore the UI")
	}
	a.deps.removeUserData = func() error { return nil }
	a.prepareForUninstall()
	if !quit {
		t.Fatal("cleanup retry did not exit")
	}
}

// TestCleanupRecoveryFailureClosesInsteadOfResuming verifies that failed resource recovery after
// cleanup closes the application.
func TestCleanupRecoveryFailureClosesInsteadOfResuming(t *testing.T) {
	a := testApp(t)
	a.deps.removeUserData = func() error { return errors.New("partial deletion") }
	a.deps.recoverAfterCleanup = func() error { return errors.New("lock unavailable") }
	a.deps.showError = func(string) {}
	a.deps.resumeWatching = func() { t.Fatal("resumed without a valid instance lock") }
	quit := false
	a.deps.quit = func() { quit = true }
	a.prepareForUninstall()
	if !quit || !a.stopping {
		t.Fatal("could not recover resources but did not close")
	}
}
