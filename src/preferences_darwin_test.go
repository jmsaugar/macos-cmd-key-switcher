//go:build darwin && cgo

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestNativePreferencesBridge exercises the Objective-C boundary in an isolated executable.
// The harness replaces UserDefaults with an in-memory object before accessing preferences.
func TestNativePreferencesBridge(t *testing.T) {
	sourceDirectory, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	contents := filepath.Join(t.TempDir(), "PreferencesTest.app", "Contents")
	executable := filepath.Join(contents, "MacOS", "preferences-test")
	if err := os.MkdirAll(filepath.Dir(executable), 0700); err != nil {
		t.Fatal(err)
	}
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>local.cmdkeyswitcher.preferences-test</string>
<key>CFBundleExecutable</key><string>preferences-test</string>
<key>CFBundlePackageType</key><string>APPL</string>
</dict></plist>`
	if err := os.WriteFile(filepath.Join(contents, "Info.plist"), []byte(plist), 0600); err != nil {
		t.Fatal(err)
	}
	compile := exec.Command("xcrun", "clang", "-x", "objective-c", "-fobjc-arc",
		"-mmacosx-version-min=13.0", "-Wall", "-Wextra", "-Wdocumentation",
		"-Wdocumentation-pedantic", "-Werror", "-Wno-unused-parameter",
		"-framework", "Foundation", "-I", sourceDirectory,
		filepath.Join(sourceDirectory, "preferences_darwin.m"),
		filepath.Join(sourceDirectory, "testdata", "preferences_bridge_test.m"), "-o", executable)
	if output, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("compile native preferences harness: %v\n%s", err, output)
	}
	if output, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("native preferences bridge: %v\n%s", err, output)
	}
}
