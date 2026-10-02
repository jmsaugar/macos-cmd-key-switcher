package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Keyboard struct {
	ID        string `json:"id"`
	Product   string `json:"product"`
	Vendor    int    `json:"vendor"`
	ProductID int    `json:"product_id"`
	BuiltIn   bool   `json:"built_in"`
	Serial    string `json:"serial,omitempty"`
}
type Config struct {
	Type string `json:"type"`
	// Keys are vendor:product or vendor:product:serial:<escaped serial>.
	KeyboardTypes map[string]string `json:"keyboard_types"`
}

const macMapping = `{"UserKeyMapping":[{"HIDKeyboardModifierMappingSrc":30064771298,"HIDKeyboardModifierMappingDst":30064771298},{"HIDKeyboardModifierMappingSrc":30064771299,"HIDKeyboardModifierMappingDst":30064771299}]}`
const winMapping = `{"UserKeyMapping":[{"HIDKeyboardModifierMappingSrc":30064771299,"HIDKeyboardModifierMappingDst":30064771298},{"HIDKeyboardModifierMappingSrc":30064771298,"HIDKeyboardModifierMappingDst":30064771299}]}`

func mappingForType(t string) string {
	if t == "win" {
		return winMapping
	}
	return macMapping
}

func defaults() Config {
	return Config{Type: "mac", KeyboardTypes: map[string]string{}}
}
func loadConfig(path string) (Config, error) {
	c := defaults()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if err = json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	if c.Type != "mac" && c.Type != "win" {
		return c, fmt.Errorf("invalid keyboard type %q", c.Type)
	}
	for id, t := range c.KeyboardTypes {
		if t != "mac" && t != "win" {
			return c, fmt.Errorf("invalid keyboard override %s: %s", id, t)
		}
	}
	return c, nil
}
func saveConfig(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), "config-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
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
func applyMapping(_ Config, t string) error {
	mapping := mappingForType(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "/usr/bin/hidutil", "property", "--set", mapping).CombinedOutput()
	if err != nil {
		return fmt.Errorf("hidutil: %w: %s", err, out)
	}
	return nil
}
