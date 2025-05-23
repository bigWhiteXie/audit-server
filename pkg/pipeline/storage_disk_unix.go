//go:build !windows
// +build !windows

package pipeline

import (
	"golang.org/x/sys/unix"
)

func (s *LocalStorage[T]) isDiskFull() bool {
	var stat unix.Statfs_t
	err := unix.Statfs(s.storageDir, &stat)
	if err != nil {
		return true
	}
	available := stat.Bavail * uint64(stat.Bsize)
	return available < minDiskSpace
}