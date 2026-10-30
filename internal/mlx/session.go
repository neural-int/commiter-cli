package mlx

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"sync"
)

// Session owns exactly one sequential Grouping/Category/Text helper cycle.
// It retains model preparation, never a response matcher. Production phases
// have independent KV state; research-only prefix plans can opt into snapshots.
type Session struct {
	client  *Client
	command *exec.Cmd
	stdin   io.WriteCloser
	frames  chan streamResult
	done    chan struct{}
	cancel  context.CancelFunc
	once    sync.Once
	mu      sync.Mutex
	count   int
	waitErr error
}

func (c *Client) BeginSession(ctx context.Context) (*Session, error) {
	if c == nil || c.Executable == "" {
		return nil, &Error{Kind: FailureStart}
	}
	child, cancel := context.WithCancel(ctx)
	cmd := c.command(child)
	cmd.Args = append(cmd.Args, "--three-phase")
	configureHelperProcess(cmd, c.grace())
	in, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		in.Close()
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		in.Close()
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		cancel()
		in.Close()
		return nil, err
	}
	s := &Session{client: c, command: cmd, stdin: in, frames: make(chan streamResult, 1), done: make(chan struct{}), cancel: cancel}
	limit := c.MaxResponseBytes
	if limit <= 0 {
		limit = DefaultMaxResponseBytes
	}
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		defer close(s.frames)
		scanner := bufio.NewScanner(out)
		scanner.Buffer(make([]byte, 4096), limit+1)
		scanner.Split(func(data []byte, atEOF bool) (int, []byte, error) {
			if i := bytes.IndexByte(data, '\n'); i >= 0 {
				return i + 1, data[:i], nil
			}
			if atEOF && len(data) > 0 {
				return 0, nil, errors.New("unterminated response")
			}
			return 0, nil, nil
		})
		for scanner.Scan() {
			data := append(append([]byte(nil), scanner.Bytes()...), '\n')
			if len(data) > limit+1 {
				s.cancel()
				break
			}
			select {
			case s.frames <- streamResult{data: data}:
			case <-child.Done():
				return
			}
		}
	}()
	go func() {
		n, _ := io.Copy(io.Discard, io.LimitReader(stderr, defaultMaxStderrBytes+1))
		if n > defaultMaxStderrBytes {
			s.cancel()
		}
	}()
	go func() { <-readDone; s.waitErr = cmd.Wait(); cleanupHelperProcess(cmd); close(s.done) }()
	return s, nil
}
func (s *Session) Close() { s.once.Do(func() { s.stdin.Close(); s.cancel(); <-s.done }) }
func (s *Session) Generate(ctx context.Context, request Request) (Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	profiles := []string{"bounded-grouping", "bounded-category", "bounded-text"}
	if s.count >= len(profiles) || request.GenerationProfile != profiles[s.count] {
		s.Close()
		return Response{}, &Error{Kind: FailureProtocol}
	}
	limit := s.client.MaxRequestBytes
	if limit <= 0 {
		limit = DefaultMaxRequestBytes
	}
	data, err := marshalRequest(request, limit)
	if err != nil {
		s.Close()
		return Response{}, &Error{Kind: FailureProtocol}
	}
	if ctx.Err() != nil {
		s.Close()
		return contextFailure(ctx.Err())
	}
	if err = writeRequest(s.stdin, data); err != nil {
		s.Close()
		return Response{}, &Error{Kind: FailureProtocol}
	}
	select {
	case <-ctx.Done():
		s.Close()
		return contextFailure(ctx.Err())
	case frame, ok := <-s.frames:
		if !ok {
			s.Close()
			return Response{}, &Error{Kind: FailureCrash}
		}
		response, err := decodeResponse(frame.data)
		if err != nil || !response.StopReason.valid() || response.GenerationProfile != request.GenerationProfile {
			s.Close()
			return response, &Error{Kind: FailureProtocol}
		}
		if !response.OK || response.StopReason != StopReasonCompleted {
			s.Close()
			return response, &Error{Kind: FailureStopState, StopReason: response.StopReason}
		}
		if response.GeneratedJSON == "" {
			s.Close()
			return response, &Error{Kind: FailureProtocol}
		}
		s.count++
		if s.count == len(profiles) {
			_ = s.stdin.Close()
			select {
			case <-ctx.Done():
				s.Close()
				return contextFailure(ctx.Err())
			case <-s.done:
			}
			if s.waitErr != nil {
				return Response{}, &Error{Kind: FailureCrash}
			}
		}
		return response, nil
	}
}
