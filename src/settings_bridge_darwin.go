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

func nativeUnregisterStartup() error {
	message := C.unregisterLoginStartup()
	if message == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(message))
	return errors.New(C.GoString(message))
}

func nativeShowError(message string) {
	value := C.CString(message)
	defer C.free(unsafe.Pointer(value))
	C.showAppError(value)
}

func stopNativeWatching() { C.endKeyboardWatching() }
func resumeNativeWatching() {
	C.beginKeyboardWatching()
	C.ensureKeyboardWatching()
}
func stopNativeApp() { C.stopApp() }
func nativeSetMenuEnabled(enabled bool) {
	value := C.int(0)
	if enabled {
		value = 1
	}
	C.setMenuEnabled(value)
}

//export appClose
func appClose() {
	if nativeApp != nil {
		nativeApp.closeApp()
	}
}

//export appPrepareForUninstall
func appPrepareForUninstall() {
	if nativeApp != nil {
		nativeApp.prepareForUninstall()
	}
}
