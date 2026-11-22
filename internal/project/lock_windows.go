//go:build windows

package project

import (
	"errors"
	"syscall"
)

// errorSharingViolation is ERROR_SHARING_VIOLATION: another handle has the
// file open without sharing it.
const errorSharingViolation = syscall.Errno(32)

// tryLock opens path shared with no one: while open, no other build can
// open it. busy says another process (or build) holds it.
func tryLock(path string) (release func(), busy bool, err error) {
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, false, err
	}
	h, err := syscall.CreateFile(name, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil,
		syscall.OPEN_ALWAYS, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		if errors.Is(err, errorSharingViolation) {
			return nil, true, nil
		}
		return nil, false, err
	}
	return func() { syscall.CloseHandle(h) }, false, nil
}
