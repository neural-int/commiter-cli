// Package mlx contains the production boundary for the private MLX helper.
//
// The helper is deliberately a child process. This package owns only the
// length-bounded, newline-delimited JSON protocol and the child lifecycle; it
// does not know about Git or planning validation. A caller must accept a
// response only when StopReasonCompleted is returned and then run its own
// domain validation.
package mlx

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

const (
	// DefaultMaxRequestBytes bounds the complete JSON request before it is
	// handed to a helper. It bounds repository content even when the helper
	// itself is unavailable.
	DefaultMaxRequestBytes = 1 << 20
	// DefaultMaxResponseBytes bounds the complete JSON response, including
	// generated content and runtime metadata.
	DefaultMaxResponseBytes = 4 << 20
	defaultMaxStderrBytes   = 64 << 10
	defaultTerminateGrace   = 250 * time.Millisecond
	defaultWaitDelay        = time.Second
)

// StopReason is the runtime-neutral completion state returned by the helper
// or synthesized at the Go boundary when the context ends.
// Only completed is a usable planning candidate.
type StopReason string

const (
	StopReasonCompleted StopReason = "completed"
	StopReasonMaxTokens StopReason = "max_tokens"
	StopReasonTimeout   StopReason = "timeout"
	StopReasonCancelled StopReason = "cancelled"
	StopReasonGrammar   StopReason = "grammar_failure"
	StopReasonInternal  StopReason = "internal_error"
)

func (reason StopReason) valid() bool {
	switch reason {
	case StopReasonCompleted, StopReasonMaxTokens, StopReasonTimeout,
		StopReasonCancelled, StopReasonGrammar, StopReasonInternal:
		return true
	default:
		return false
	}
}

// Request is the complete helper input. Message content is sent only through
// the child's stdin; this package never writes it to logs or diagnostics.
type Request struct {
	Schema        json.RawMessage `json:"schema"`
	Messages      []llm.Message   `json:"messages"`
	ContextTokens int             `json:"context_tokens"`
	OutputTokens  int             `json:"output_tokens"`
	// Model is a display identifier. ModelPath is an already prepared local
	// model directory; the helper never resolves Model through a network
	// downloader.
	Model     string `json:"model,omitempty"`
	ModelPath string `json:"model_path,omitempty"`
}

// Response is the machine-readable helper output. GeneratedJSON is accepted
// by a caller only when StopReason is StopReasonCompleted.
type Response struct {
	OK            bool       `json:"ok"`
	StopReason    StopReason `json:"stop_reason"`
	GeneratedJSON string     `json:"generated_json,omitempty"`
	Model         string     `json:"model,omitempty"`
	Runtime       string     `json:"runtime,omitempty"`
	ErrorClass    string     `json:"error_class,omitempty"`
}

// FailureKind identifies failures without exposing helper diagnostics or
// repository content in an error message.
type FailureKind string

const (
	FailureStart     FailureKind = "start"
	FailureTimeout   FailureKind = "timeout"
	FailureCancelled FailureKind = "cancelled"
	FailureCrash     FailureKind = "crash"
	FailureProtocol  FailureKind = "protocol"
	FailureOversized FailureKind = "oversized"
	FailureStopState FailureKind = "stop_state"
)

// Error is returned for lifecycle, protocol, and non-completed stop-state
// failures. It intentionally omits helper stderr and all request/response
// content.
type Error struct {
	Kind       FailureKind
	StopReason StopReason
	ExitCode   int
	Cause      error
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	switch err.Kind {
	case FailureStart:
		return "MLX helper could not be started"
	case FailureTimeout:
		return "MLX helper timed out"
	case FailureCancelled:
		return "MLX helper was cancelled"
	case FailureCrash:
		return "MLX helper exited unexpectedly"
	case FailureOversized:
		return "MLX helper protocol message exceeded its limit"
	case FailureStopState:
		return "MLX helper did not complete generation"
	case FailureProtocol:
		return "MLX helper returned an invalid protocol response"
	default:
		return "MLX helper failed"
	}
}

func (err *Error) Unwrap() error { return err.Cause }

// Client launches one helper per Generate call. A helper must not be reused:
// this prevents one malformed request or stale model state from contaminating
// a later planning run.
type Client struct {
	Executable       string
	MaxRequestBytes  int
	MaxResponseBytes int
	TerminateGrace   time.Duration
	Command          commandFactory
}

// NewClient returns a client with the production protocol limits.
func NewClient(executable string) *Client {
	return &Client{
		Executable:       executable,
		MaxRequestBytes:  DefaultMaxRequestBytes,
		MaxResponseBytes: DefaultMaxResponseBytes,
		TerminateGrace:   defaultTerminateGrace,
	}
}

// Generate sends one bounded request and waits for one bounded response.
// Context cancellation and timeout are classified separately from a helper
// crash. A response with any stop reason other than completed is rejected even
// when its partial JSON happens to parse.
func (client *Client) Generate(ctx context.Context, request Request) (Response, error) {
	if client == nil || strings.TrimSpace(client.Executable) == "" {
		return Response{}, &Error{Kind: FailureStart, Cause: errors.New("helper executable is required")}
	}
	maxRequest := client.MaxRequestBytes
	if maxRequest <= 0 {
		maxRequest = DefaultMaxRequestBytes
	}
	maxResponse := client.MaxResponseBytes
	if maxResponse <= 0 {
		maxResponse = DefaultMaxResponseBytes
	}
	requestData, err := marshalRequest(request, maxRequest)
	if err != nil {
		kind := FailureProtocol
		if errors.Is(err, errMessageTooLarge) {
			kind = FailureOversized
		}
		return Response{}, &Error{Kind: kind, Cause: err}
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return contextFailure(ctxErr)
	}

	command := client.command(ctx)
	configureHelperProcess(command, client.grace())
	stdin, err := command.StdinPipe()
	if err != nil {
		return Response{}, &Error{Kind: FailureStart, Cause: err}
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return Response{}, &Error{Kind: FailureStart, Cause: err}
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		return Response{}, &Error{Kind: FailureStart, Cause: err}
	}
	if err := command.Start(); err != nil {
		_ = stdin.Close()
		if ctxErr := ctx.Err(); ctxErr != nil {
			return contextFailure(ctxErr)
		}
		return Response{}, &Error{Kind: FailureStart, Cause: err}
	}

	stdoutResult := make(chan streamResult, 1)
	stderrResult := make(chan streamResult, 1)
	overflow := make(chan struct{}, 1)
	go readBoundedStream(stdout, maxResponse, stdoutResult, overflow)
	go readBoundedStream(stderr, defaultMaxStderrBytes, stderrResult, overflow)
	stopOverflowWatcher := make(chan struct{})
	go func() {
		select {
		case <-overflow:
			_ = cancelHelperProcess(command, client.grace())
		case <-stopOverflowWatcher:
		}
	}()
	writeErr := writeRequest(stdin, requestData)
	_ = stdin.Close()
	stdoutData := <-stdoutResult
	// stderr is intentionally observed only to drain the pipe. Its content is
	// never included in an Error or returned to the caller.
	stderrData := <-stderrResult
	waitErr := command.Wait()
	close(stopOverflowWatcher)
	cleanupHelperProcess(command)

	if ctx.Err() != nil {
		return contextFailure(ctx.Err())
	}
	if errors.Is(stdoutData.err, errMessageTooLarge) {
		return Response{}, &Error{Kind: FailureOversized, Cause: stdoutData.err}
	}
	if errors.Is(stderrData.err, errMessageTooLarge) {
		return Response{}, &Error{Kind: FailureOversized, Cause: stderrData.err}
	}
	if waitErr != nil {
		return Response{}, &Error{Kind: FailureCrash, ExitCode: exitCode(waitErr), Cause: waitErr}
	}
	if stdoutData.err != nil {
		return Response{}, &Error{Kind: FailureProtocol, Cause: stdoutData.err}
	}
	if writeErr != nil {
		return Response{}, &Error{Kind: FailureProtocol, Cause: writeErr}
	}
	response, err := decodeResponse(stdoutData.data)
	if err != nil {
		return Response{}, &Error{Kind: FailureProtocol, Cause: err}
	}
	if !response.StopReason.valid() {
		return response, &Error{Kind: FailureProtocol, StopReason: response.StopReason}
	}
	if !response.OK || response.StopReason != StopReasonCompleted {
		return response, &Error{Kind: FailureStopState, StopReason: response.StopReason}
	}
	if strings.TrimSpace(response.GeneratedJSON) == "" {
		return response, &Error{Kind: FailureProtocol, Cause: errors.New("completed response has no generated JSON")}
	}
	return response, nil
}

func contextFailure(cause error) (Response, error) {
	kind := FailureCancelled
	reason := StopReasonCancelled
	if errors.Is(cause, context.DeadlineExceeded) {
		kind = FailureTimeout
		reason = StopReasonTimeout
	}
	return Response{StopReason: reason}, &Error{Kind: kind, StopReason: reason, Cause: cause}
}

type commandFactory func(context.Context, string, ...string) *exec.Cmd

func (client *Client) command(ctx context.Context) *exec.Cmd {
	if client.Command != nil {
		return client.Command(ctx, client.Executable)
	}
	return exec.CommandContext(ctx, client.Executable)
}

func (client *Client) grace() time.Duration {
	if client.TerminateGrace > 0 {
		return client.TerminateGrace
	}
	return defaultTerminateGrace
}

var errMessageTooLarge = errors.New("message exceeds configured limit")

func marshalRequest(request Request, limit int) ([]byte, error) {
	if len(request.Schema) == 0 || !json.Valid(request.Schema) {
		return nil, errors.New("request schema must be valid JSON")
	}
	data, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	if len(data)+1 > limit {
		return nil, errMessageTooLarge
	}
	return append(data, '\n'), nil
}

func writeRequest(writer io.Writer, data []byte) error {
	if _, err := writer.Write(data); err != nil {
		return err
	}
	return nil
}

type streamResult struct {
	data []byte
	err  error
}

func readBoundedStream(reader io.Reader, limit int, result chan<- streamResult, overflow chan<- struct{}) {
	data := bytes.NewBuffer(nil)
	buffered := bufio.NewReaderSize(reader, 32*1024)
	chunk := make([]byte, 32*1024)
	for {
		n, err := buffered.Read(chunk)
		if n > 0 {
			if data.Len()+n > limit {
				select {
				case overflow <- struct{}{}:
				default:
				}
				// Continue draining so a helper that writes diagnostics cannot
				// deadlock before Wait reaps it.
				_, _ = io.Copy(io.Discard, buffered)
				result <- streamResult{data: data.Bytes(), err: errMessageTooLarge}
				return
			}
			_, _ = data.Write(chunk[:n])
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				result <- streamResult{data: data.Bytes()}
			} else {
				result <- streamResult{data: data.Bytes(), err: err}
			}
			return
		}
	}
}

func decodeResponse(data []byte) (Response, error) {
	reader := bufio.NewReader(bytes.NewReader(data))
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return Response{}, errors.New("response must end with a newline")
	}
	if len(bytes.TrimSpace(line[:len(line)-1])) == 0 {
		return Response{}, errors.New("response is empty")
	}
	var response Response
	decoder := json.NewDecoder(bytes.NewReader(bytes.TrimSpace(line[:len(line)-1])))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&response); err != nil {
		return Response{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return Response{}, errors.New("response contains trailing JSON")
	}
	if extra, err := reader.ReadByte(); err == nil || extra != 0 {
		return Response{}, errors.New("response contains trailing bytes")
	}
	return response, nil
}

func exitCode(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}
