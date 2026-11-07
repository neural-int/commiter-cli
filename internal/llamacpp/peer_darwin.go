package llamacpp

import (
	"net"
	"os"
	"syscall"
)

func localOwner(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Getuid())
}

func peerPID(connection net.Conn) int {
	unix, ok := connection.(*net.UnixConn)
	if !ok {
		return -1
	}
	raw, err := unix.SyscallConn()
	if err != nil {
		return -1
	}
	pid := -1
	// macOS sys/un.h: SOL_LOCAL=0, LOCAL_PEERPID=0x002.
	if err := raw.Control(func(fd uintptr) {
		value, err := syscall.GetsockoptInt(int(fd), 0, 0x002)
		if err == nil {
			pid = value
		}
	}); err != nil {
		return -1
	}
	return pid
}
