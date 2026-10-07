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

// nativeUnregisterStartup removes login startup registration, treating an absent
// registration as success. It converts and frees any native error message.
func nativeUnregisterStartup() error {
	message := C.unregisterLoginStartup()
	if message == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(message))
	return errors.New(C.GoString(message))
}

// nativeShowError displays message in a modal warning and returns when dismissed.
func nativeShowError(message string) {
	value := C.CString(message)
	defer C.free(unsafe.Pointer(value))
	C.showAppError(value)
}

// stopNativeWatching releases native keyboard notification resources.
func stopNativeWatching() { C.endKeyboardWatching() }

// resumeNativeWatching reenables device watching and attempts to install notifications.
func resumeNativeWatching() {
	C.beginKeyboardWatching()
	C.ensureKeyboardWatching()
}

// stopNativeApp requests that the Cocoa event loop return.
func stopNativeApp() { C.stopApp() }

// nativeSetMenuEnabled enables or disables the top-level menu items.
func nativeSetMenuEnabled(enabled bool) {
	value := C.int(0)
	if enabled {
		value = 1
	}
	C.setMenuEnabled(value)
}

// appClose forwards the menu close action on the main thread.
//
//export appClose
func appClose() {
	if nativeApp != nil {
		nativeApp.closeApp()
	}
}

// appPrepareForUninstall forwards confirmed cleanup on the main thread.
//
//export appPrepareForUninstall
func appPrepareForUninstall() {
	if nativeApp != nil {
		nativeApp.prepareForUninstall()
	}
}
