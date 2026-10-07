package main

import (
	"path/filepath"
	"testing"
)

// TestDetection verifies that device types and saved overrides select the expected mapping.
//
// The parameter t runs the test and reports assertion failures.
//
// Failures are reported through t.
func TestDetection(t *testing.T) {
	c := defaults()
	cases := []struct {
		name    string
		devices []Keyboard
		want    string
	}{
		{"none", nil, "mac"},
		{"built in", []Keyboard{{Vendor: 1, BuiltIn: true}}, "mac"},
		{"apple", []Keyboard{{Vendor: 1452}}, "mac"},
		{"windows", []Keyboard{{Vendor: 123}}, "win"},
		{"mixed", []Keyboard{{Vendor: 1452}, {Vendor: 123}}, "win"},
	}
	for _, tc := range cases {
		// Callback checks the detected mapping for the current table case.
		//
		// The parameter t reports failures for this subtest.
		t.Run(tc.name, func(t *testing.T) {
			if got := detectedType(tc.devices, c); got != tc.want {
				t.Fatalf("got %s want %s", got, tc.want)
			}
		})
	}
	c.KeyboardTypes["123:456"] = "mac"
	if detectedType([]Keyboard{{Vendor: 123, ProductID: 456}}, c) != "mac" {
		t.Fatal("override ignored")
	}
}

// TestFingerprint verifies that device signatures ignore enumeration order and distinguish
// devices.
//
// The parameter t runs the test and reports assertion failures.
//
// Failures are reported through t.
func TestFingerprint(t *testing.T) {
	a := Keyboard{ID: "a", Vendor: 1}
	b := Keyboard{ID: "b", Vendor: 2}
	if fingerprint([]Keyboard{a, b}) != fingerprint([]Keyboard{b, a}) {
		t.Fatal("enumeration order changed fingerprint")
	}
	if fingerprint([]Keyboard{a}) == fingerprint([]Keyboard{b}) {
		t.Fatal("device change ignored")
	}
}

// TestUnitOverridesAndModelFallback verifies that unit preferences override model defaults and
// persist without affecting other units.
//
// The parameter t runs the test and reports assertion failures.
//
// Failures are reported through t.
func TestUnitOverridesAndModelFallback(t *testing.T) {
	a := Keyboard{Vendor: 1234, ProductID: 5678, Serial: "A /:1"}
	b := a
	b.Serial = "B"
	c := defaults()
	c.KeyboardTypes[modelKey(a)] = "win"
	rememberKeyboards(&c, []Keyboard{a}, "mac")
	if detectedType([]Keyboard{a}, c) != "mac" || detectedType([]Keyboard{b}, c) != "win" {
		t.Fatal("unit override did not take precedence or leaked to another unit")
	}
	if c.KeyboardTypes[modelKey(a)] != "win" {
		t.Fatal("unit selection changed model override")
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := saveConfig(path, c); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if detectedType([]Keyboard{a}, loaded) != "mac" {
		t.Fatal("unit preference lost across reload")
	}
	noSerial := a
	noSerial.Serial = "  "
	rememberKeyboards(&c, []Keyboard{noSerial}, "mac")
	if detectedType([]Keyboard{b}, c) != "mac" {
		t.Fatal("model fallback was not saved")
	}
	internal := Keyboard{Vendor: 42, ProductID: 9, Serial: "internal", BuiltIn: true}
	rememberKeyboards(&c, []Keyboard{internal}, "win")
	if c.KeyboardTypes[unitKey(internal)] != "" {
		t.Fatal("built-in keyboard remembered")
	}
}
