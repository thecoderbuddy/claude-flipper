//go:build windows

// Package lock provides a simple cross-process file lock.
package lock

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
)

const (
	lockfileExclusiveLock = 0x00000002
	lockfileFailImmediately = 0x00000001
)

// Lock holds a held file lock.
type Lock struct {
	f *os.File
}

// Acquire creates (or opens) the file at path and places an exclusive LockFileEx on it.
// It blocks until the lock is available.
func Acquire(path string) (*Lock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("create lock dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	ol := new(windows.Overlapped)
	r1, _, err := procLockFileEx.Call(
		uintptr(f.Fd()),
		uintptr(lockfileExclusiveLock), // exclusive, blocking
		0,
		1, 0,
		uintptr(unsafe.Pointer(ol)),
	)
	if r1 == 0 {
		_ = f.Close()
		return nil, fmt.Errorf("LockFileEx: %w", err)
	}
	return &Lock{f: f}, nil
}

// Release unlocks and closes the lock file.
func (l *Lock) Release() {
	ol := new(windows.Overlapped)
	_, _, _ = procUnlockFileEx.Call(
		uintptr(l.f.Fd()),
		0,
		1, 0,
		uintptr(unsafe.Pointer(ol)),
	)
	_ = l.f.Close()
}
