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

// nativeApp binds the one AppKit application to the Go callbacks in both bridges.
// It is read and written only on the main OS thread. Tests use independent App
// instances without replacing this binding.
var nativeApp *App

// runNativeApp binds a to native callbacks until the Cocoa event loop returns.
// It must be called on the main OS thread.
func runNativeApp(a *App) {
	nativeApp = a
	defer func() { nativeApp = nil }()
	C.runApp()
}

// scheduleNativeRetry schedules a one-shot callback with a delay in seconds.
func scheduleNativeRetry(seconds int) { C.scheduleRetry(C.double(seconds)) }

// cancelNativeRetry cancels the native retry timer.
func cancelNativeRetry() { C.cancelRetry() }

// nativeUpdateMenu sends the applied mode and optional error text to Cocoa.
func nativeUpdateMenu(modeValue, errorValue string) {
	mode := C.CString(modeValue)
	defer C.free(unsafe.Pointer(mode))
	message := C.CString(errorValue)
	defer C.free(unsafe.Pointer(message))
	C.updateMenu(mode, message)
}

// connectedKeyboards reads and decodes keyboard registry metadata.
// Enumeration and JSON decoding failures return an error.
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

// appTick handles startup and debounced HID service notifications on the main thread.
//
//export appTick
func appTick() {
	if nativeApp != nil {
		nativeApp.keyboardChanged()
	}
}

// appSwitch forwards a manual menu switch to the active app on the main thread.
//
//export appSwitch
func appSwitch() {
	if nativeApp != nil {
		nativeApp.switchKeyboard()
	}
}

// appWake forwards a workspace wake notification on the main thread.
//
//export appWake
func appWake() {
	if nativeApp != nil {
		nativeApp.keyboardWake()
	}
}

// appRetry forwards a one-shot retry timer callback on the main thread.
//
//export appRetry
func appRetry() {
	if nativeApp != nil {
		nativeApp.retryAutomatic()
	}
}

// appMappingComplete receives an already-enqueued worker result on the main thread
// and forwards it to the active app. The worker must send before signaling this callback.
//
//export appMappingComplete
func appMappingComplete() {
	if nativeApp != nil {
		nativeApp.finishMapping(<-nativeApp.mappingResults)
	}
}

// notifyMappingComplete can be called by a worker to enqueue a native main-thread
// completion callback after sending its result.
func notifyMappingComplete() { C.notifyMappingComplete() }
