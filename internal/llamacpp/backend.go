// Package llamacpp contains the runtime-independent portions of the Issue 190
// adapter. No upstream build has passed the qualification gate yet. In
// particular, a bundle manifest or a schema-valid response is not evidence
// that LLGuidance failed closed during generation.
package llamacpp

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

const QualificationMessage = "llamacpp is not enabled: no unmodified pinned llama.cpp build has passed LLGuidance fail-closed, UDS peer identity, parent-death cleanup, Metal and three-phase qualification (see docs/runtime/llamacpp.md)"

var ErrUnqualifiedRuntime = errors.New(QualificationMessage)

// Backend deliberately has no external endpoint, executable path, environment
// override or feature flag that could bypass the release qualification gate.
type Backend struct{}

var _ llm.CycleBackend = (*Backend)(nil)

func (*Backend) BeginCycle(ctx context.Context) (llm.OptionsBackend, func() error, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	return nil, nil, ErrUnqualifiedRuntime
}

func (b *Backend) Chat(ctx context.Context, messages []llm.Message, schema json.RawMessage) (llm.Response, error) {
	return b.ChatWithOptions(ctx, messages, schema, llm.Options{})
}

func (*Backend) ChatWithOptions(ctx context.Context, _ []llm.Message, _ json.RawMessage, _ llm.Options) (llm.Response, error) {
	if err := ctx.Err(); err != nil {
		return llm.Response{}, err
	}
	return llm.Response{}, ErrUnqualifiedRuntime
}
