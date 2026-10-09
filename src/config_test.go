package main

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
)

// memoryPreferences keeps test preferences isolated from macOS's persistent domains.
type memoryPreferences struct {
	data                        []byte
	readErr, writeErr, clearErr error
}

// read returns an independent copy of the test snapshot.
func (p *memoryPreferences) read() ([]byte, error) {
	return append([]byte(nil), p.data...), p.readErr
}

// write replaces the snapshot unless a submission failure is configured.
func (p *memoryPreferences) write(data []byte) error {
	if p.writeErr != nil {
		return p.writeErr
	}
	p.data = append([]byte(nil), data...)
	return nil
}

// clear resets test preferences unless a cleanup failure is configured.
func (p *memoryPreferences) clear() error {
	if p.clearErr != nil {
		return p.clearErr
	}
	p.data = nil
	return nil
}

// TestConfigRoundTrip verifies that mode and overrides survive a storage reload and clear.
func TestConfigRoundTrip(t *testing.T) {
	preferences := &memoryPreferences{}
	c, err := loadConfig(preferences)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(c, defaults()) {
		t.Fatalf("missing preferences did not load defaults: %+v", c)
	}
	c.Type = "win"
	c.KeyboardTypes["1:2"] = "mac"
	c.KeyboardTypes["1:2:serial:unit+A"] = "win"
	if err = saveConfig(preferences, c); err != nil {
		t.Fatal(err)
	}
	got, err := loadConfig(preferences)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, c) {
		t.Fatalf("config did not round trip: %+v", got)
	}
	got.KeyboardTypes["1:2"] = "win"
	unchanged, err := loadConfig(preferences)
	if err != nil || unchanged.KeyboardTypes["1:2"] != "mac" {
		t.Fatal("mutating loaded preferences changed stored data")
	}
	if err := preferences.clear(); err != nil {
		t.Fatal(err)
	}
	reset, err := loadConfig(preferences)
	if err != nil || !reflect.DeepEqual(reset, defaults()) {
		t.Fatalf("clearing preferences did not restore defaults: %+v, %v", reset, err)
	}
}

// TestConfigRejectsInvalidPreferences verifies malformed data and unknown modes fail validation.
func TestConfigRejectsInvalidPreferences(t *testing.T) {
	for _, data := range []string{
		`not JSON`, `[]`, `{"type":123}`, `{"type":"other"}`,
		`{"keyboard_types":{"1:2":false}}`, `{"keyboard_types":{"1:2":"other"}}`,
	} {
		t.Run(data, func(t *testing.T) {
			if _, err := loadConfig(&memoryPreferences{data: []byte(data)}); err == nil {
				t.Fatal("accepted invalid preferences")
			}
		})
	}
	for _, c := range []Config{{Type: "other"}, {Type: "mac", KeyboardTypes: map[string]string{"1:2": "other"}}} {
		preferences := &memoryPreferences{data: []byte(`{"type":"win"}`)}
		before := append([]byte(nil), preferences.data...)
		if err := saveConfig(preferences, c); err == nil || !bytes.Equal(before, preferences.data) {
			t.Fatal("invalid configuration changed stored preferences")
		}
	}
}

// TestConfigPropagatesStorageErrors verifies bridge failures reach the caller.
func TestConfigPropagatesStorageErrors(t *testing.T) {
	failure := errors.New("storage unavailable")
	if _, err := loadConfig(&memoryPreferences{readErr: failure}); !errors.Is(err, failure) {
		t.Fatalf("read error lost: %v", err)
	}
	if err := saveConfig(&memoryPreferences{writeErr: failure}, defaults()); !errors.Is(err, failure) {
		t.Fatalf("write error lost: %v", err)
	}
}

// TestConfigNormalizesEmptyOverrides verifies nil maps become property-list-compatible dictionaries.
func TestConfigNormalizesEmptyOverrides(t *testing.T) {
	preferences := &memoryPreferences{}
	if err := saveConfig(preferences, Config{Type: "win"}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(preferences.data, []byte(`"keyboard_types":{}`)) {
		t.Fatalf("nil overrides were not encoded as a dictionary: %s", preferences.data)
	}
}
