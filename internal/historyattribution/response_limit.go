package historyattribution

import (
	"bufio"
	"errors"
)

const researchResponseLimit = 300000000

var errResponseLimit = errors.New("response_size_limit")

type responseLimitContext struct{}

// Bound the line before appending/decoding it. The reader may prefetch at most
// its buffer size beyond the accepted prefix; the decoded JSON never exceeds cap.
func readLimitedResponse(reader *bufio.Reader, limit int) ([]byte, error) {
	if limit <= 0 {
		return reader.ReadBytes('\n')
	}
	var result []byte
	for {
		fragment, err := reader.ReadSlice('\n')
		if len(fragment) > limit-len(result) {
			return nil, errResponseLimit
		}
		result = append(result, fragment...)
		if err == bufio.ErrBufferFull {
			continue
		}
		return result, err
	}
}
