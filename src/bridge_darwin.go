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

func runNativeApp(a *App) {
	nativeApp = a
	defer func() { nativeApp = nil }()
	C.runApp()
}
func scheduleNativeRetry(seconds int) { C.scheduleRetry(C.double(seconds)) }
func cancelNativeRetry()              { C.cancelRetry() }

func nativeUpdateMenu(modeValue, errorValue string) {
	mode := C.CString(modeValue)
	defer C.free(unsafe.Pointer(mode))
	message := C.CString(errorValue)
	defer C.free(unsafe.Pointer(message))
	C.updateMenu(mode, message)
}
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

//export appTick
func appTick() {
	if nativeApp != nil {
		nativeApp.keyboardChanged()
	}
}

//export appSwitch
func appSwitch() {
	if nativeApp != nil {
		nativeApp.switchKeyboard()
	}
}

//export appWake
func appWake() {
	if nativeApp != nil {
		nativeApp.keyboardWake()
	}
}

//export appRetry
func appRetry() {
	if nativeApp != nil {
		nativeApp.retryAutomatic()
	}
}

//export appMappingComplete
func appMappingComplete() {
	if nativeApp != nil {
		nativeApp.finishMapping(<-nativeApp.mappingResults)
	}
}

func notifyMappingComplete() { C.notifyMappingComplete() }
