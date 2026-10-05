//go:build darwin && cgo

package main

import (
	"os"
	"path/filepath"
	"testing"
)

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
