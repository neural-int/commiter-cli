//go:build !darwin && !linux

package ggufmodel

import (
	"context"
	"os"
)

func owned(os.FileInfo) bool                                      { return false }
func openNoFollow(string, int, os.FileMode) (*os.File, error)     { return nil, ErrUnsafe }
func acquire(context.Context, string, bool) (func() error, error) { return nil, ErrUnsafe }
