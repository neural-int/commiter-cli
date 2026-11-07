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
	if err := raw.Control(func(fd uintptr) {
		cred, err := syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
		if err == nil {
			pid = int(cred.Pid)
		}
	}); err != nil {
		return -1
	}
	return pid
}
