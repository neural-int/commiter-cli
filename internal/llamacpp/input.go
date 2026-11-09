package llamacpp

import (
	"context"
	"errors"
	"math/bits"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

const ContextTokens = 65536
const OutputTokens = 4096

// preparedInput retains the exact token sequence admitted by native token
// counting. Generation must use it without templating or tokenizing it again.
type preparedInput struct {
	TokenIDs               []int
	SelectedHistoryIndices []int
	PreparationCalls       int
}

type tokenizeSelection func(context.Context, []int) ([]int, error)

// prepareInput mirrors bounded HistoryInputPreparation in the MLX helper. Byte
// estimates shortlist whole records only; every admitted input is tokenized.
// An empty estimate falls back to individual native measurements within the
// same three-candidate bound. Byte overflow can shrink optional history too;
// all other preparation errors end the cycle immediately.
func prepareInput(ctx context.Context, byteCounts []int, prepare tokenizeSelection) (preparedInput, error) {
	const limit = ContextTokens - OutputTokens
	all := make([]int, len(byteCounts))
	totalBytes := uint64(0)
	for i, count := range byteCounts {
		if count <= 0 || uint64(count) > ^uint64(0)-totalBytes {
			return preparedInput{}, errors.New("llamacpp: invalid history byte count")
		}
		totalBytes += uint64(count)
		all[i] = i
	}
	calls := 0
	run := func(indices []int) (preparedInput, error) {
		if err := ctx.Err(); err != nil {
			return preparedInput{}, err
		}
		tokens, err := prepare(ctx, append([]int{}, indices...))
		calls++
		if err != nil {
			return preparedInput{}, err
		}
		for _, token := range tokens {
			if token < 0 {
				return preparedInput{}, errors.New("llamacpp: invalid native tokens")
			}
		}
		return preparedInput{TokenIDs: append([]int(nil), tokens...), SelectedHistoryIndices: append([]int{}, indices...), PreparationCalls: calls}, nil
	}
	full, err := run(all)
	if err != nil && !errors.Is(err, errRequestTooLarge) {
		return preparedInput{}, err
	}
	if err == nil && len(full.TokenIDs) <= limit {
		return full, nil
	}
	if len(all) == 0 {
		if err != nil {
			return preparedInput{}, err
		}
		return preparedInput{}, llm.ErrInputTooLarge
	}
	fullByteOverflow := errors.Is(err, errRequestTooLarge)
	base, err := run([]int{})
	if err != nil {
		return preparedInput{}, err
	}
	if len(base.TokenIDs) > limit {
		return preparedInput{}, llm.ErrInputTooLarge
	}
	remaining := uint64(limit - len(base.TokenIDs))
	estimate := func(bytes uint64, observed int) uint64 {
		// Called only after overflow: observed > remaining, so quotient fits.
		hi, lo := bits.Mul64(bytes, remaining)
		q, _ := bits.Div64(hi, lo, uint64(observed))
		return q
	}
	budget := totalBytes / 2
	if !fullByteOverflow {
		budget = estimate(totalBytes, len(full.TokenIDs)-len(base.TokenIDs))
	}
	measuredSingles := make([]bool, len(all))
	for attempt := 0; attempt < 3; attempt++ {
		indices := []int{}
		selectedBytes := uint64(0)
		for i, count := range byteCounts {
			if uint64(count) <= budget-selectedBytes {
				indices = append(indices, i)
				selectedBytes += uint64(count)
			}
		}
		if len(indices) == 0 || len(indices) == 1 && measuredSingles[indices[0]] {
			// Bytes cannot prove that a record does not fit in tokens. Try
			// the next unmeasured whole record in the original priority order.
			indices = nil
			for i, measured := range measuredSingles {
				if !measured {
					indices = []int{i}
					selectedBytes = uint64(byteCounts[i])
					break
				}
			}
			if len(indices) == 0 {
				break
			}
		}
		if len(indices) == 1 {
			measuredSingles[indices[0]] = true
		}
		candidate, err := run(indices)
		if err != nil && !errors.Is(err, errRequestTooLarge) {
			return preparedInput{}, err
		}
		if err == nil && len(candidate.TokenIDs) <= limit {
			return candidate, nil
		}
		budget = selectedBytes / 2
		if err == nil {
			budget = min(budget, estimate(selectedBytes, len(candidate.TokenIDs)-len(base.TokenIDs)))
		}
	}
	base.PreparationCalls = calls
	return base, nil
}
