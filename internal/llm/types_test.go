package llm

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestIsRetryableUsesBackendNeutralClassification(t *testing.T) {
	if !IsRetryable(context.DeadlineExceeded) {
		t.Fatal("deadline should be retryable")
	}
	if IsRetryable(context.Canceled) || IsRetryable(errors.New("invalid response")) {
		t.Fatal("non-retryable error was accepted")
	}
	if !IsRetryable(retryableError{}) {
		t.Fatal("backend retry classification was ignored")
	}
	if !IsRetryable(&net.DNSError{IsTimeout: true}) {
		t.Fatal("timeout error should be retryable")
	}
}

func TestAliasesExposeTheRuntimeNeutralContract(t *testing.T) {
	var response ChatResponse
	var options ChatOptions
	var capability CapabilityResult
	if response.Backend != "" || options.ContextTokens != 0 || capability.StructuredOutput {
		t.Fatal("aliases do not preserve the neutral types")
	}
}

type retryableError struct{}

func (retryableError) Error() string   { return "temporary backend failure" }
func (retryableError) Retryable() bool { return true }
