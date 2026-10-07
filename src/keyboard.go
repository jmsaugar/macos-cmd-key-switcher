package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

type Keyboard struct {
	ID        string `json:"id"`
	Product   string `json:"product"`
	Vendor    int    `json:"vendor"`
	ProductID int    `json:"product_id"`
	BuiltIn   bool   `json:"built_in"`
	Serial    string `json:"serial,omitempty"`
}

// modelKey builds the preference key shared by a keyboard model.
//
// The parameter k supplies the vendor and product identifiers.
// It returns the vendor:product key.
func modelKey(k Keyboard) string { return fmt.Sprintf("%d:%d", k.Vendor, k.ProductID) }

// unitKey builds a preference key for a keyboard with a nonblank serial number.
//
// The parameter k supplies model identifiers and the serial number.
// It returns the model key with an escaped serial suffix, or an empty string if no serial is
// available.
func unitKey(k Keyboard) string {
	serial := strings.TrimSpace(k.Serial)
	if serial == "" {
		return ""
	}
	return modelKey(k) + ":serial:" + url.QueryEscape(serial)
}

// rememberKeyboards stores the selected mode for external keyboards, preferring unit keys over
// model keys.
//
// The parameter c is the configuration to mutate; devices is the snapshot; mode is the selected mapping.
func rememberKeyboards(c *Config, devices []Keyboard, mode string) {
	for _, k := range devices {
		if k.BuiltIn {
			continue
		}
		if c.KeyboardTypes == nil {
			c.KeyboardTypes = map[string]string{}
		}
		key := unitKey(k)
		if key == "" {
			key = modelKey(k)
		}
		c.KeyboardTypes[key] = mode
	}
}

// detectedType selects a mapping using unit preferences, model preferences, and vendor defaults.
//
// The parameter devices is the keyboard snapshot; c supplies saved preferences.
// It returns win if any external keyboard prefers it; otherwise mac.
func detectedType(devices []Keyboard, c Config) string {
	for _, k := range devices {
		if k.BuiltIn {
			continue
		}
		t := c.KeyboardTypes[unitKey(k)]
		if t == "" {
			t = c.KeyboardTypes[modelKey(k)]
		}
		if t == "" {
			if k.Vendor == 1452 {
				t = "mac"
			} else {
				t = "win"
			}
		}
		if t == "win" {
			return "win"
		}
	}
	return "mac"
}

// fingerprint creates a stable device signature independent of enumeration order.
//
// The parameter devices is the keyboard snapshot to identify.
// It returns a JSON string of sorted device identity keys.
func fingerprint(devices []Keyboard) string {
	keys := make([]string, 0, len(devices))
	for _, k := range devices {
		keys = append(keys, fmt.Sprintf("%s:%d:%d:%t:%s", k.ID, k.Vendor, k.ProductID, k.BuiltIn, unitKey(k)))
	}
	sort.Strings(keys)
	b, _ := json.Marshal(keys)
	return string(b)
}
