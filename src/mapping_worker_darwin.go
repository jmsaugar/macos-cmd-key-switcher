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
var activeMapping, pendingMapping *mappingRequest
var mappingResults = make(chan mappingResult, 1)
var startMapping = launchMapping
var signalMappingComplete = notifyMappingComplete

func requestedType() string {
	if pendingMapping != nil {
		return pendingMapping.mode
	}
	if activeMapping != nil {
		return activeMapping.mode
	}
	return config.Type
}
func requestMapping(mode string, devices []Keyboard, automatic bool) {
	request := mappingRequest{mode, append([]Keyboard(nil), devices...), automatic, observedDevices}
	lastDevices = request.signature
	if activeMapping != nil {
		// A newer request replaces waiting work, never the currently running command.
		pendingMapping = &request
		return
	}
	activeMapping = &request
	startMapping(request)
}
func launchMapping(request mappingRequest) {
	execute := executeMapping
	// The current executor takes Config but hidutil only needs the requested mode.
	snapshot := Config{Type: config.Type}
	signal := signalMappingComplete
	go func() {
		mappingResults <- mappingResult{request, execute(snapshot, request.mode)}
		signal()
	}()
}

// Called on the main thread after dispatching completion through Cocoa.
func finishMapping(result mappingResult) {
	next := pendingMapping
	activeMapping, pendingMapping = nil, nil
	if result.err != nil {
		lastError = result.err.Error()
		log.Print(lastError)
		if next == nil && result.request.automatic {
			retryMode = result.request.mode
			queueRetry()
		}
	} else {
		config.Type = result.request.mode
		// Superseded manual choices must not teach a preference the user replaced.
		if next == nil {
			rememberKeyboards(&config, result.request.devices, result.request.mode)
		}
		if err := saveConfig(configPath, config); err != nil {
			lastError = "Mapping applied, but config could not be saved: " + err.Error()
			log.Print(lastError)
		} else {
			lastError = ""
		}
		if next == nil {
			retryMode = ""
		}
	}
	refreshMenu()
	if next != nil {
		activeMapping = next
		startMapping(*next)
	}
}
