//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -x objective-c -fobjc-arc -mmacosx-version-min=13.0
#cgo LDFLAGS: -mmacosx-version-min=13.0 -framework Cocoa -framework IOKit -framework CoreFoundation -framework ServiceManagement
#include <stdlib.h>
#include "native.h"
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"unsafe"
)

// All Go/native conversions and callbacks live here.
// AppKit supports one running application. Only the bridge holds this pointer;
// tests create independent App instances and do not replace it.
var nativeApp *App

// runNativeApp binds the application to native callbacks and runs the Cocoa event loop.
//
// The receiver a is the application receiving callbacks.
// It returns after the native event loop stops.
func runNativeApp(a *App) {
	nativeApp = a
	// Callback clears the active native application binding after the event loop exits.
	defer func() { nativeApp = nil }()
	C.runApp()
}

// scheduleNativeRetry schedules a native one-shot retry timer.
//
// The parameter seconds is the delay before the retry callback.
func scheduleNativeRetry(seconds int) { C.scheduleRetry(C.double(seconds)) }

// cancelNativeRetry cancels the native retry timer.
func cancelNativeRetry() { C.cancelRetry() }

// nativeUpdateMenu passes the current mapping and error to the Cocoa menu.
//
// The parameter modeValue is the applied mode; errorValue is the error text or an empty string.
func nativeUpdateMenu(modeValue, errorValue string) {
	mode := C.CString(modeValue)
	defer C.free(unsafe.Pointer(mode))
	message := C.CString(errorValue)
	defer C.free(unsafe.Pointer(message))
	C.updateMenu(mode, message)
}

// connectedKeyboards reads and decodes the native keyboard registry snapshot.
// It returns connected keyboards and nil, or nil and an enumeration or JSON decoding error.
func connectedKeyboards() ([]Keyboard, error) {
	raw := C.keyboardJSON()
	if raw == nil {
		return nil, fmt.Errorf("could not enumerate keyboards")
	}
	defer C.free(unsafe.Pointer(raw))
	var devices []Keyboard
	if err := json.Unmarshal([]byte(C.GoString(raw)), &devices); err != nil {
		return nil, err
	}
	return devices, nil
}

// appTick forwards a native device notification to the active application on the main thread.
//
//export appTick
func appTick() {
	if nativeApp != nil {
		nativeApp.keyboardChanged()
	}
}

// appSwitch forwards a native manual switch action to the active application on the main thread.
//
//export appSwitch
func appSwitch() {
	if nativeApp != nil {
		nativeApp.switchKeyboard()
	}
}

// appWake forwards a native wake notification to the active application on the main thread.
//
//export appWake
func appWake() {
	if nativeApp != nil {
		nativeApp.keyboardWake()
	}
}

// appRetry forwards a native retry timer callback to the active application on the main thread.
//
//export appRetry
func appRetry() {
	if nativeApp != nil {
		nativeApp.retryAutomatic()
	}
}

// appMappingComplete receives a worker result and finishes its mapping on the main thread.
//
// Waits for the signaled result when an application is active.
//
//export appMappingComplete
func appMappingComplete() {
	if nativeApp != nil {
		nativeApp.finishMapping(<-nativeApp.mappingResults)
	}
}

// notifyMappingComplete dispatches a worker completion notification to the native main thread.
func notifyMappingComplete() { C.notifyMappingComplete() }
