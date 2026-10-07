//go:build darwin && cgo

package main

/*
#include <stdlib.h>
#include "native.h"
*/
import "C"
import (
	"errors"
	"unsafe"
)

// nativeUnregisterStartup removes the native launch-at-login registration.
// It returns nil on success, or the error reported by ServiceManagement.
func nativeUnregisterStartup() error {
	message := C.unregisterLoginStartup()
	if message == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(message))
	return errors.New(C.GoString(message))
}

// nativeShowError displays a native modal error alert.
//
// The parameter message is the error text to show.
// It returns when the alert is dismissed.
func nativeShowError(message string) {
	value := C.CString(message)
	defer C.free(unsafe.Pointer(value))
	C.showAppError(value)
}

// stopNativeWatching releases native keyboard notification resources.
func stopNativeWatching() { C.endKeyboardWatching() }

// resumeNativeWatching reenables keyboard watching and attempts to install native notifications.
func resumeNativeWatching() {
	C.beginKeyboardWatching()
	C.ensureKeyboardWatching()
}

// stopNativeApp requests that the Cocoa event loop stop.
func stopNativeApp() { C.stopApp() }

// nativeSetMenuEnabled sets whether top-level native menu items accept actions.
//
// The parameter enabled selects the menu enabled state.
func nativeSetMenuEnabled(enabled bool) {
	value := C.int(0)
	if enabled {
		value = 1
	}
	C.setMenuEnabled(value)
}

// appClose forwards the native close action to the active application on the main thread.
//
//export appClose
func appClose() {
	if nativeApp != nil {
		nativeApp.closeApp()
	}
}

// appPrepareForUninstall forwards confirmed native cleanup to the active application on the main
// thread.
//
//export appPrepareForUninstall
func appPrepareForUninstall() {
	if nativeApp != nil {
		nativeApp.prepareForUninstall()
	}
}
