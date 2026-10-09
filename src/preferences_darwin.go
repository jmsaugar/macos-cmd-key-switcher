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

// nativePreferences accesses the running app bundle's UserDefaults domain.
// Runtime calls belong to the main OS thread, like other App state transitions.
type nativePreferences struct{}

// read retrieves the configuration dictionary as a temporary JSON bridge value.
func (nativePreferences) read() ([]byte, error) {
	raw := C.readPreferences()
	if raw == nil {
		return nil, errors.New("could not read app preferences; run the app bundle and check its stored configuration")
	}
	defer C.free(unsafe.Pointer(raw))
	return []byte(C.GoString(raw)), nil
}

// write submits a validated configuration to UserDefaults without waiting for disk persistence.
func (nativePreferences) write(data []byte) error {
	value := C.CString(string(data))
	defer C.free(unsafe.Pointer(value))
	message := C.writePreferences(value)
	if message == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(message))
	return errors.New(C.GoString(message))
}

// clear removes the app's persistent preferences domain through UserDefaults.
func (nativePreferences) clear() error {
	message := C.clearPreferences()
	if message == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(message))
	return errors.New(C.GoString(message))
}
