//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package dedupe

import (
	"os"
	"syscall"
)

type fileIdentity struct {
	device uint64
	inode  uint64
}

func getFileIdentity(_ string, info os.FileInfo) (fileIdentity, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fileIdentity{}, false
	}
	return fileIdentity{device: uint64(stat.Dev), inode: uint64(stat.Ino)}, true
}
