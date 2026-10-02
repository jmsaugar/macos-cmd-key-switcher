package main

import (
	"path/filepath"
	"testing"
)

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
func TestConfigRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	c, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	c.Type = "win"
	c.KeyboardTypes["1:2"] = "mac"
	if err = saveConfig(path, c); err != nil {
		t.Fatal(err)
	}
	got, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != "win" || got.KeyboardTypes["1:2"] != "mac" {
		t.Fatal("config did not round trip")
	}
}
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
