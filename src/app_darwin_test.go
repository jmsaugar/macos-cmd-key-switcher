//go:build darwin && cgo

package main

import "testing"

// TestAppsOwnIndependentStateAndDependencies verifies that application instances keep state and
// dependencies independent.
func TestAppsOwnIndependentStateAndDependencies(t *testing.T) {
	t.Parallel()
	a, b := testApp(t), testApp(t)
	a.updateDevices([]Keyboard{{ID: "external", Vendor: 1234}})
	if a.config.Type != "win" || b.config.Type != "mac" {
		t.Fatal("selection leaked between instances")
	}
	a.switchKeyboard()
	if b.config.KeyboardTypes["1234:0"] != "" {
		t.Fatal("preferences leaked between instances")
	}
	saved := false
	b.deps.saveConfig = func(string, Config) error { saved = true; return nil }
	b.requestMapping("win", nil, false)
	if !saved || a.config.Type != "mac" {
		t.Fatal("dependency or state changes leaked between instances")
	}
}

// TestAppCopiesInitialPreferences verifies that an application copies its initial preference map.
func TestAppCopiesInitialPreferences(t *testing.T) {
	t.Parallel()
	config := defaults()
	config.KeyboardTypes["1:2"] = "win"
	a := newApp("unused", config)
	config.KeyboardTypes["1:2"] = "mac"
	if a.config.KeyboardTypes["1:2"] != "win" {
		t.Fatal("App shares its caller's preference map")
	}
	a.config.KeyboardTypes["3:4"] = "mac"
	if config.KeyboardTypes["3:4"] != "" {
		t.Fatal("App mutated its caller's preferences")
	}
}
