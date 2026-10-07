//go:build darwin && cgo

package main

import (
	"os"
	"path/filepath"
	"syscall"
)

type instanceLock struct {
	path string
	file *os.File
}

// acquireInstanceLock opens path and acquires an exclusive, nonblocking file lock.
// The caller must close the returned lock to release it.
func acquireInstanceLock(path string) (*instanceLock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		return nil, err
	}
	return &instanceLock{path, file}, nil
}

// restore reacquires the lock pathname if cleanup removed or replaced its file.
// An open descriptor can still lock an unlinked file, so the pathname must be
// locked again before the app resumes after partial cleanup.
func (lock *instanceLock) restore() error {
	held, err := lock.file.Stat()
	if err != nil {
		return err
	}
	if current, err := os.Stat(lock.path); err == nil && os.SameFile(held, current) {
		return nil
	}
	replacement, err := acquireInstanceLock(lock.path)
	if err != nil {
		return err
	}
	lock.file.Close()
	lock.file = replacement.file
	return nil
}

// close releases the instance lock by closing its file.
func (lock *instanceLock) close() { lock.file.Close() }
