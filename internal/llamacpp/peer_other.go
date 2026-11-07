//go:build !darwin && !linux

package llamacpp

import (
	"net"
	"os"
)

func localOwner(os.FileInfo) bool { return false }
func peerPID(net.Conn) int        { return -1 }
