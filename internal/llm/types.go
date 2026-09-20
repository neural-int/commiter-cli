// Package llm defines the runtime-neutral contract used by planning and
// backend adapters. It intentionally contains no provider-specific protocol
// fields or lifecycle operations.
package llm

import (
	"context"
	"encoding/json"
	"errors"
)

// Message is a role/content chat message shared by all local backends.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// TelemetryAvailability distinguishes an omitted backend measurement from a
// measurement whose value is zero. Backends must set a field when the
// corresponding value was observed.
type TelemetryAvailability struct {
	TotalDuration      bool
	LoadDuration       bool
	PromptEvalCount    bool
	PromptEvalDuration bool
	EvalCount          bool
	EvalDuration       bool
}

// Any reports whether at least one telemetry field was observed.
func (availability TelemetryAvailability) Any() bool {
	return availability.TotalDuration || availability.LoadDuration ||
		availability.PromptEvalCount || availability.PromptEvalDuration ||
		availability.EvalCount || availability.EvalDuration
}

// Response contains generated content and privacy-safe numeric telemetry.
// Duration fields are elapsed nanoseconds, matching time.Duration's underlying
// unit without exposing a provider-specific duration type.
type Response struct {
	Backend            string
	Model              string
	Content            string
	TotalDuration      int64
	LoadDuration       int64
	PromptEvalCount    int
	PromptEvalDuration int64
	EvalCount          int
	EvalDuration       int64
	Availability       TelemetryAvailability
}

// ChatResponse is a descriptive alias for backend implementations.
type ChatResponse = Response

// Options contains provider-neutral generation limits.
type Options struct {
	ContextTokens int
	OutputTokens  int
}

// ChatOptions is a descriptive alias for callers that use chat terminology.
type ChatOptions = Options

// Capability describes capabilities probed without changing backend state.
type Capability struct {
	StructuredOutput bool
	ThinkingDisabled bool
}

// CapabilityResult is a descriptive alias for capability probes.
type CapabilityResult = Capability

// CapabilityBackend is an optional extension for probing backend capabilities
// without changing backend state.
type CapabilityBackend interface {
	Backend
	ProbeCapabilities(context.Context) (Capability, error)
}

// Backend is the planning-facing contract implemented by backend adapters.
type Backend interface {
	Chat(context.Context, []Message, json.RawMessage) (Response, error)
}

// OptionsBackend is an optional extension for callers that provide explicit
// context and output budgets.
type OptionsBackend interface {
	Backend
	ChatWithOptions(context.Context, []Message, json.RawMessage, Options) (Response, error)
}

// Retryable marks a backend error that is safe to retry once.
type Retryable interface {
	Retryable() bool
}

// IsRetryable reports whether a failed request may consume the shared retry
// budget. Backend adapters can classify transport failures without exposing
// provider-specific error types to callers.
func IsRetryable(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	var retryable Retryable
	if errors.As(err, &retryable) {
		return retryable.Retryable()
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var timeout interface{ Timeout() bool }
	return errors.As(err, &timeout) && timeout.Timeout()
}
