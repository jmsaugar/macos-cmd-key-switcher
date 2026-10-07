package main

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

const macMapping = `{"UserKeyMapping":[{"HIDKeyboardModifierMappingSrc":30064771298,"HIDKeyboardModifierMappingDst":30064771298},{"HIDKeyboardModifierMappingSrc":30064771299,"HIDKeyboardModifierMappingDst":30064771299}]}`
const winMapping = `{"UserKeyMapping":[{"HIDKeyboardModifierMappingSrc":30064771299,"HIDKeyboardModifierMappingDst":30064771298},{"HIDKeyboardModifierMappingSrc":30064771298,"HIDKeyboardModifierMappingDst":30064771299}]}`

// mappingForType selects the hidutil modifier mapping JSON for a keyboard mode.
//
// The parameter t is the requested mode.
// It returns the Windows mapping for win, otherwise the mac mapping.
func mappingForType(t string) string {
	if t == "win" {
		return winMapping
	}
	return macMapping
}

// applyMapping applies the selected mapping with hidutil under a five-second timeout.
//
// The parameter t is the requested keyboard mode.
// It returns nil on success, or a command error including hidutil output.
func applyMapping(t string) error {
	mapping := mappingForType(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "/usr/bin/hidutil", "property", "--set", mapping).CombinedOutput()
	if err != nil {
		return fmt.Errorf("hidutil: %w: %s", err, out)
	}
	return nil
}
