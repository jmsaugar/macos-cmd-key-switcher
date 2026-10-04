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

func modelKey(k Keyboard) string { return fmt.Sprintf("%d:%d", k.Vendor, k.ProductID) }
func unitKey(k Keyboard) string {
	serial := strings.TrimSpace(k.Serial)
	if serial == "" {
		return ""
	}
	return modelKey(k) + ":serial:" + url.QueryEscape(serial)
}
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
func fingerprint(devices []Keyboard) string {
	keys := make([]string, 0, len(devices))
	for _, k := range devices {
		keys = append(keys, fmt.Sprintf("%s:%d:%d:%t:%s", k.ID, k.Vendor, k.ProductID, k.BuiltIn, unitKey(k)))
	}
	sort.Strings(keys)
	b, _ := json.Marshal(keys)
	return string(b)
}
