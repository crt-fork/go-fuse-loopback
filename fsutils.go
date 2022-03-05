package loopbackfs

import (
	"os"
	"syscall"

	"github.com/protosam/go-fuse-loopback/pkg/device"
)

func syscallMode(i os.FileMode) (o uint32) {
	o |= uint32(i.Perm())
	if i&os.ModeSetuid != 0 {
		o |= syscall.S_ISUID
	}
	if i&os.ModeSetgid != 0 {
		o |= syscall.S_ISGID
	}
	if i&os.ModeSticky != 0 {
		o |= syscall.S_ISVTX
	}
	return
}

func syscallMakeDev(path string) (int, error) {
	stat := syscall.Stat_t{}
	if err := syscall.Stat(path, &stat); err != nil {
		return -1, err
	}

	device.Makedev(device.Major(uint64(stat.Rdev)), device.Minor(uint64(stat.Rdev)))
	return 0, nil
}
