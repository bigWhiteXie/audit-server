//go:build windows
// +build windows

package pipeline

import (
	"syscall"
	"unsafe"
)

func (s *LocalStorage) isDiskFull() bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceExW := kernel32.NewProc("GetDiskFreeSpaceExW")

	lpDirectoryName, _ := syscall.UTF16PtrFromString(s.storageDir)
	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes int64

	ret, _, _ := getDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(lpDirectoryName)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
	)
	if ret == 0 {
		return true
	}
	return uint64(freeBytesAvailable) < minDiskSpace
}
