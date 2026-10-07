package main

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// macMapping maps left Option and left Command to themselves.
const macMapping = `{"UserKeyMapping":[{"HIDKeyboardModifierMappingSrc":30064771298,"HIDKeyboardModifierMappingDst":30064771298},{"HIDKeyboardModifierMappingSrc":30064771299,"HIDKeyboardModifierMappingDst":30064771299}]}`

// winMapping swaps left Command and left Option.
const winMapping = `{"UserKeyMapping":[{"HIDKeyboardModifierMappingSrc":30064771299,"HIDKeyboardModifierMappingDst":30064771298},{"HIDKeyboardModifierMappingSrc":30064771298,"HIDKeyboardModifierMappingDst":30064771299}]}`

// mappingForType returns winMapping for "win" and macMapping otherwise.
func mappingForType(t string) string {
	if t == "win" {
		return winMapping
	}
	return macMapping
}

// applyMapping runs hidutil directly with a five-second timeout.
// On failure, the returned error includes the command error and captured output.
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
