//go:build darwin && cgo

package main

// App owns mutable application state. Runtime state transitions belong to the
// main OS thread; concurrent method calls are not safe.
// Workers receive copied requests and return results through mappingResults.
type App struct {
	config                Config
	configPath, lastError string
	// Device observations and automatic reapplication.
	lastDevices, observedDevices string
	forceRefresh                 bool
	// Applied mode is config.Type; these requests represent desired future modes.
	activeMapping, pendingMapping *mappingRequest
	mappingResults                chan mappingResult
	retries                       retryBudget
	retryMode                     string
	deps                          appDependencies
	// Shutdown waits for the active command before exiting or deleting files.
	stopping, cleaningUp bool
	logsPath             string
}

// appDependencies binds native operations and command execution per App instance.
// Tests replace these operations without changing package globals.
type appDependencies struct {
	readKeyboards          func() ([]Keyboard, error)
	executeMapping         func(string) error
	saveConfig             func(string, Config) error
	updateMenu             func(string, string)
	scheduleAutomaticRetry func(int)
	cancelAutomaticRetry   func()
	startMapping           func(mappingRequest)
	signalMappingComplete  func()
	unregisterStartup      func() error
	removeUserData         func() error
	recoverAfterCleanup    func() error
	stopWatching           func()
	resumeWatching         func()
	setMenuEnabled         func(bool)
	quit                   func()
	showError              func(string)
}

// newApp copies the initial preferences and binds production dependencies.
// The caller must bind cleanup recovery to the startup resources before running the app.
func newApp(path string, config Config) *App {
	// Take ownership of preferences instead of sharing the caller's map.
	preferences := make(map[string]string, len(config.KeyboardTypes))
	for key, mode := range config.KeyboardTypes {
		preferences[key] = mode
	}
	config.KeyboardTypes = preferences
	a := &App{config: config, configPath: path, mappingResults: make(chan mappingResult, 1)}
	a.deps = appDependencies{
		readKeyboards: connectedKeyboards, executeMapping: applyMapping,
		saveConfig: saveConfig, updateMenu: nativeUpdateMenu,
		scheduleAutomaticRetry: scheduleNativeRetry, cancelAutomaticRetry: cancelNativeRetry,
		startMapping: a.launchMapping, signalMappingComplete: notifyMappingComplete,
		unregisterStartup: nativeUnregisterStartup, removeUserData: a.removeUserData,
		recoverAfterCleanup: func() error { return nil }, // Bound to startup resources by main.
		stopWatching:        stopNativeWatching, resumeWatching: resumeNativeWatching,
		setMenuEnabled: nativeSetMenuEnabled, quit: stopNativeApp, showError: nativeShowError,
	}
	return a
}

// refreshMenu displays the applied mode and latest error.
func (a *App) refreshMenu() { a.deps.updateMenu(a.config.Type, a.lastError) }

// close cancels retries and discards pending mappings. It does not wait for
// an active command, release the instance lock, or stop the native event loop.
func (a *App) close() {
	a.deps.cancelAutomaticRetry()
	a.pendingMapping = nil
}
