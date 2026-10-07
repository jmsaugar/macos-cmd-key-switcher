//go:build darwin && cgo

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestInstanceLockSurvivesCleanupFailure verifies that restoring a lock after partial cleanup
// still excludes another instance.
//
// The parameter t runs the test and reports assertion failures.
//
// Failures are reported through t.
func TestInstanceLockSurvivesCleanupFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json.lock")
	lock, err := acquireInstanceLock(path)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.close()
	for _, remove := range []bool{false, true} {
		if remove {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		}
		if err := lock.restore(); err != nil {
			t.Fatal(err)
		}
		duplicate, err := acquireInstanceLock(path)
		if err == nil {
			duplicate.close()
			t.Fatal("another instance could acquire the lock")
		}
	}
}
