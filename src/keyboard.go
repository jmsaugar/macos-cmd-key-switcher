package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// Keyboard holds metadata for one primary keyboard HID service in IORegistry.
// A physical device may expose multiple services with the same vendor, product, and serial.
type Keyboard struct {
	ID        string `json:"id"`               // Transient IORegistry entry ID.
	Product   string `json:"product"`          // Device-supplied name or a fallback label.
	Vendor    int    `json:"vendor"`           // HID vendor ID; zero if unavailable.
	ProductID int    `json:"product_id"`       // HID product ID; zero if unavailable.
	BuiltIn   bool   `json:"built_in"`         // Inferred from Built-In or SPI/i2c transport.
	Serial    string `json:"serial,omitempty"` // Device-supplied serial, possibly absent or nonunique.
}

// modelKey returns the decimal vendor:product preference key for k.
func modelKey(k Keyboard) string { return fmt.Sprintf("%d:%d", k.Vendor, k.ProductID) }

// unitKey returns a model key with a URL-escaped, trimmed serial suffix.
// It returns an empty string when k has no nonblank serial number. Device-supplied
// serials are not guaranteed to be unique or stable.
func unitKey(k Keyboard) string {
	serial := strings.TrimSpace(k.Serial)
	if serial == "" {
		return ""
	}
	return modelKey(k) + ":serial:" + url.QueryEscape(serial)
}

// rememberKeyboards records mode for every external keyboard in devices,
// using a serial-based key when available and a model key otherwise.
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

// detectedType returns "win" if any external keyboard is classified as Windows,
// and "mac" otherwise, including when no external keyboards are connected.
// Serial-based overrides take precedence over model overrides, then vendor defaults.
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

// fingerprint returns an enumeration-order-independent signature of devices.
// It includes transient registry IDs so reconnecting a device can change the signature;
// it is not a persistent device identifier.
func fingerprint(devices []Keyboard) string {
	keys := make([]string, 0, len(devices))
	for _, k := range devices {
		keys = append(keys, fmt.Sprintf("%s:%d:%d:%t:%s", k.ID, k.Vendor, k.ProductID, k.BuiltIn, unitKey(k)))
	}
	sort.Strings(keys)
	b, _ := json.Marshal(keys)
	return string(b)
}
