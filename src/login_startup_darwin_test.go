//go:build darwin && cgo

package main

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// TestNativeLoginStartupUnregistration exercises native status and error handling.
// The isolated harness replaces the login service before any registration calls.
func TestNativeLoginStartupUnregistration(t *testing.T) {
	sourceDirectory, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(t.TempDir(), "login-startup-test")
	compile := exec.Command("xcrun", "clang", "-x", "objective-c", "-fobjc-arc",
		"-mmacosx-version-min=13.0", "-Wall", "-Wextra", "-Wdocumentation",
		"-Wdocumentation-pedantic", "-Werror", "-Wno-unused-parameter",
		"-framework", "Cocoa", "-framework", "ServiceManagement", "-I", sourceDirectory,
		filepath.Join(sourceDirectory, "login_startup_darwin.m"),
		filepath.Join(sourceDirectory, "testdata", "login_startup_bridge_test.m"), "-o", executable)
	if output, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("compile native login startup harness: %v\n%s", err, output)
	}
	if output, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("native login startup unregistration: %v\n%s", err, output)
	}
}
