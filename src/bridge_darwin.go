//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework IOKit -framework CoreFoundation
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
func runNativeApp()                   { C.runApp() }
func scheduleNativeRetry(seconds int) { C.scheduleRetry(C.double(seconds)) }
func cancelNativeRetry()              { C.cancelRetry() }

func refreshMenu() {
	mode := C.CString(config.Type)
	defer C.free(unsafe.Pointer(mode))
	message := C.CString(lastError)
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
func appTick() { keyboardChanged() }

//export appSwitch
func appSwitch() { switchKeyboard() }

//export appWake
func appWake() { keyboardWake() }

//export appRetry
func appRetry() { retryAutomatic() }
