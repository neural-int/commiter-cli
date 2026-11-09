package llamacpp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var errTransport = errors.New("llamacpp: local transport validation failed")
var errRequestTooLarge = errors.New("llamacpp: local request exceeds byte limit")

// localTransport is an internal building block, unreachable through Backend
// until an upstream revision has passed qualification. Each connection checks
// the owned socket inode and OS peer PID before sending any request bytes.
type localTransport struct {
	client    *http.Client
	transport *http.Transport
}

func newLocalTransport(socket string, pid int) (*localTransport, error) {
	if pid <= 0 || !filepath.IsAbs(socket) || len(socket) >= 90 {
		return nil, errTransport
	}
	directory, err := os.Lstat(filepath.Dir(socket))
	if err != nil || !directory.IsDir() || directory.Mode().Perm() != 0700 || !localOwner(directory) {
		return nil, errTransport
	}
	identity, err := socketIdentity(socket)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{
		Proxy: nil, DisableKeepAlives: true, MaxConnsPerHost: 1,
		MaxResponseHeaderBytes: 32 << 10, ResponseHeaderTimeout: 10 * time.Second,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			before, err := socketIdentity(socket)
			if err != nil || !os.SameFile(identity, before) {
				return nil, errTransport
			}
			connection, err := (&net.Dialer{}).DialContext(ctx, "unix", socket)
			if err != nil {
				return nil, errTransport
			}
			after, err := socketIdentity(socket)
			if err != nil || !os.SameFile(identity, after) || peerPID(connection) != pid {
				_ = connection.Close()
				return nil, errTransport
			}
			return connection, nil
		},
	}
	return &localTransport{transport: transport, client: &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return errTransport }}}, nil
}

func socketIdentity(socket string) (os.FileInfo, error) {
	info, err := os.Lstat(socket)
	if err != nil || info.Mode()&os.ModeSocket == 0 || info.Mode().Perm() != 0600 || !localOwner(info) {
		return nil, errTransport
	}
	return info, nil
}

func (t *localTransport) close() { t.transport.CloseIdleConnections() }

func (t *localTransport) request(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	valid := method == http.MethodGet && (path == "/health" || path == "/props") || method == http.MethodPost && (path == "/apply-template" || path == "/tokenize" || path == "/completion")
	if !valid {
		return nil, errTransport
	}
	if len(body) > MaxRequestBytes {
		return nil, errRequestTooLarge
	}
	request, err := http.NewRequestWithContext(ctx, method, "http://commiter"+path, bytes.NewReader(body))
	if err != nil {
		return nil, errTransport
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := t.client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errTransport
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errTransport
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, MaxResponseBytes+1))
	if err != nil || len(data) > MaxResponseBytes {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errTransport
	}
	return data, nil
}
