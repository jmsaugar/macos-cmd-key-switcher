//go:build darwin && cgo

package main

import "log"

// mappingRequest is a device snapshot and desired mode copied for worker execution.
type mappingRequest struct {
	mode      string
	devices   []Keyboard
	automatic bool
	signature string
}

// mappingResult carries command completion back to the main thread.
type mappingResult struct {
	request mappingRequest
	err     error
}

// requestedType returns the pending mode, then the active mode, or the applied
// mode when neither request exists.
func (a *App) requestedType() string {
	if a.pendingMapping != nil {
		return a.pendingMapping.mode
	}
	if a.activeMapping != nil {
		return a.activeMapping.mode
	}
	return a.config.Type
}

// requestMapping copies the device snapshot and queues mode unless stopping.
// Only one command runs at a time; a newer request replaces waiting work.
// The automatic flag enables retries for an unsuperseded command failure.
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

// launchMapping captures the request and worker dependencies on the main thread,
// then executes the command in a goroutine. The worker sends its result before
// signaling completion and never reads mutable App state.
func (a *App) launchMapping(request mappingRequest) {
	execute := a.deps.executeMapping
	signal := a.deps.signalMappingComplete
	results := a.mappingResults
	go func() {
		results <- mappingResult{request, execute(request.mode)}
		signal()
	}()
}

// finishMapping handles a worker result on the main thread.
// During normal close it preserves the active result; during cleanup it skips
// persistence so completed work cannot recreate deleted configuration.
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

// completeMapping updates the applied mode after command success and saves config.
// Only an unsuperseded request can teach device preferences or schedule a command
// retry. A save failure is reported without rolling back the applied mapping.
// If another request is pending, it starts after this result is processed.
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
