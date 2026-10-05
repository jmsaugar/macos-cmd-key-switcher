//go:build darwin && cgo

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Called only after the UI confirms the cleanup action.
func (a *App) prepareForUninstall() {
	if a.stopping {
		return
	}
	// Keep the bundle and data available if unregistering fails.
	if err := a.deps.unregisterStartup(); err != nil {
		a.deps.showError("Could not disable launch at login: " + err.Error())
		return
	}
	a.beginShutdown(true)
}

func (a *App) closeApp() { a.beginShutdown(false) }

func (a *App) beginShutdown(cleanup bool) {
	if a.stopping {
		return
	}
	a.stopping, a.cleaningUp = true, cleanup
	a.close()
	a.deps.stopWatching()
	a.deps.setMenuEnabled(false)
	if a.activeMapping == nil {
		a.finishShutdown()
	}
}

func (a *App) finishShutdown() {
	if a.cleaningUp {
		if err := a.deps.removeUserData(); err != nil {
			if recoveryErr := a.deps.recoverAfterCleanup(); recoveryErr != nil {
				a.deps.showError(fmt.Sprintf("Could not remove app data: %v\nCould not restore app resources: %v\nThe app will close. Launch at login is disabled.", err, recoveryErr))
				a.deps.quit()
				return
			}
			// Stay open so the user can see the failure and retry cleanup.
			a.stopping, a.cleaningUp = false, false
			a.deps.resumeWatching()
			a.deps.setMenuEnabled(true)
			a.lastError = "Could not remove app data: " + err.Error()
			log.Print(a.lastError)
			a.refreshMenu()
			a.deps.showError(a.lastError + "\nLaunch at login is disabled. Some files may already have been removed.")
			return
		}
	}
	a.deps.quit()
}

func (a *App) removeUserData() error {
	for _, path := range []string{a.logsPath, filepath.Dir(a.configPath)} {
		if path == "" {
			continue
		}
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	return nil
}
