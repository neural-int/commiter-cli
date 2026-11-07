//go:build darwin || linux

package ggufmodel

import (
	"context"
	"errors"
	"os"
	"sync"
	"syscall"
	"time"
)

func owned(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Getuid())
}

func openNoFollow(path string, flags int, mode os.FileMode) (*os.File, error) {
	return os.OpenFile(path, flags|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, mode)
}

// The stable inode is never unlinked. Closing the FD releases the flock even
// after process death, and the descriptor is not inherited by a child.
func acquire(ctx context.Context, path string, exclusive bool) (func() error, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	f, err := openNoFollow(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, ErrUnsafe
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || !owned(info) {
		_ = f.Close()
		return nil, ErrUnsafe
	}
	op := syscall.LOCK_SH
	if exclusive {
		op = syscall.LOCK_EX
	}
	for {
		if err := ctx.Err(); err != nil {
			_ = f.Close()
			return nil, errors.Join(ErrBusy, err)
		}
		err = syscall.Flock(int(f.Fd()), op|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if err != syscall.EWOULDBLOCK && err != syscall.EAGAIN {
			_ = f.Close()
			return nil, ErrUnsafe
		}
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
	var once sync.Once
	var closeErr error
	return func() error { once.Do(func() { closeErr = f.Close() }); return closeErr }, nil
}
