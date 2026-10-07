//go:build darwin && cgo

package main

// App owns mutable application state. Its methods run on the main OS thread.
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

// newApp creates an application with independent preferences and native dependencies.
//
// The parameter path is the config file location; config supplies the initial preferences.
// It returns the initialized application.
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
		// Callback provides a successful recovery placeholder until main binds startup resources.
		// It returns nil to simulate success.
		recoverAfterCleanup: func() error { return nil }, // Bound to startup resources by main.
		stopWatching:        stopNativeWatching, resumeWatching: resumeNativeWatching,
		setMenuEnabled: nativeSetMenuEnabled, quit: stopNativeApp, showError: nativeShowError,
	}
	return a
}

// refreshMenu updates the menu with the applied mode and latest error.
//
// The receiver a is the application whose state is displayed.
func (a *App) refreshMenu() { a.deps.updateMenu(a.config.Type, a.lastError) }

// close cancels retries and discards pending mapping work.
//
// The receiver a is the application to stop scheduling work for.
func (a *App) close() {
	a.deps.cancelAutomaticRetry()
	a.pendingMapping = nil
}
