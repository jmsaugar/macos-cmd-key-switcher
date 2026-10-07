//go:build darwin && cgo

package main

import "log"

type mappingRequest struct {
	mode      string
	devices   []Keyboard
	automatic bool
	signature string
}
type mappingResult struct {
	request mappingRequest
	err     error
}

// These queue fields belong to the main thread. Only the result channel crosses
// threads; workers never read application state or invoke Cocoa directly.

// requestedType finds the latest requested mode, falling back to the applied mode.
//
// The receiver a supplies pending, active, and applied mapping state.
// It returns the pending mode, active mode, or configured mode, in that order.
func (a *App) requestedType() string {
	if a.pendingMapping != nil {
		return a.pendingMapping.mode
	}
	if a.activeMapping != nil {
		return a.activeMapping.mode
	}
	return a.config.Type
}

// requestMapping starts a mapping or replaces the pending request while a command is active.
//
// The parameter mode is the target; devices supplies preferences to remember; automatic enables retries on
// failure; a owns the queue.
//
// Requests are ignored during shutdown.
func (a *App) requestMapping(mode string, devices []Keyboard, automatic bool) {
	if a.stopping {
		return
	}
	request := mappingRequest{mode, append([]Keyboard(nil), devices...), automatic, a.observedDevices}
	a.lastDevices = request.signature
	if a.activeMapping != nil {
		// A newer request replaces waiting work, never the currently running command.
		a.pendingMapping = &request
		return
	}
	a.activeMapping = &request
	a.deps.startMapping(request)
}

// launchMapping runs a copied mapping request in a worker and signals its result.
//
// The parameter request contains the mode, devices, origin, and signature; a supplies worker dependencies.
//
// Completion arrives asynchronously through mappingResults.
func (a *App) launchMapping(request mappingRequest) {
	execute := a.deps.executeMapping
	signal := a.deps.signalMappingComplete
	results := a.mappingResults
	// Callback executes the copied request, sends its result, and signals completion.
	go func() {
		results <- mappingResult{request, execute(request.mode)}
		signal()
	}()
}

// Called on the main thread after dispatching completion through Cocoa.
// finishMapping handles a completed mapping on the main thread, including shutdown coordination.
//
// The parameter result contains the executed request and its error; a owns application state.
func (a *App) finishMapping(result mappingResult) {
	if a.stopping {
		a.activeMapping = nil
		if !a.cleaningUp {
			// A normal close preserves the result of an already-running selection.
			a.pendingMapping = nil
			a.completeMapping(result)
		} else if result.err == nil {
			// Reflect the applied mode if cleanup later fails and the app stays open.
			a.config.Type = result.request.mode
		}
		a.finishShutdown()
		return
	}
	a.completeMapping(result)
}

// completeMapping records a mapping result, persists successful choices, and starts pending work.
//
// The parameter result contains the executed request and its error; a owns preferences and the queue.
//
// Errors are logged and displayed in the menu.
func (a *App) completeMapping(result mappingResult) {
	next := a.pendingMapping
	a.activeMapping, a.pendingMapping = nil, nil
	if result.err != nil {
		a.lastError = result.err.Error()
		log.Print(a.lastError)
		if next == nil && result.request.automatic {
			a.retryMode = result.request.mode
			a.queueRetry()
		}
	} else {
		a.config.Type = result.request.mode
		// Superseded manual choices must not teach a preference the user replaced.
		if next == nil {
			rememberKeyboards(&a.config, result.request.devices, result.request.mode)
		}
		if err := a.deps.saveConfig(a.configPath, a.config); err != nil {
			a.lastError = "Mapping applied, but config could not be saved: " + err.Error()
			log.Print(a.lastError)
		} else {
			a.lastError = ""
		}
		if next == nil {
			a.retryMode = ""
		}
	}
	a.refreshMenu()
	if next != nil {
		a.activeMapping = next
		a.deps.startMapping(*next)
	}
}
