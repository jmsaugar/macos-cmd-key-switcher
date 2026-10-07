//go:build darwin && cgo

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Called only after the UI confirms the cleanup action.
// prepareForUninstall disables login startup and begins cleanup after UI confirmation.
//
// The receiver a is the application to clean up.
//
// Startup errors are shown and leave the application running.
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

// closeApp begins normal shutdown while preserving user data.
//
// The receiver a is the application to close.
func (a *App) closeApp() { a.beginShutdown(false) }

// beginShutdown stops new work and waits for any active mapping before finishing shutdown.
//
// The parameter cleanup selects whether to remove user data; a is the application to stop.
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

// finishShutdown completes cleanup and exits, or restores operation after a recoverable cleanup
// failure.
//
// The receiver a holds the shutdown state and resource dependencies.
//
// Failures are displayed through the application error handler.
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

// removeUserData removes the application log and configuration directories.
//
// The receiver a supplies the log and config paths.
// It returns nil on success, or the first removal error annotated with its path.
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
