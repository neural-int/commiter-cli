package historyattribution

import (
	"bufio"
	"context"
	"strings"
	"testing"
	"time"
)

func TestResponseLimitBoundary(t *testing.T) {
	for _, tc := range []struct {
		body     string
		limit    int
		exceeded bool
	}{{"1234\n", 5, false}, {"12345\n", 5, true}, {strings.Repeat("x", 10000) + "\n", 9999, true}} {
		b, e := readLimitedResponse(bufio.NewReaderSize(strings.NewReader(tc.body), 16), tc.limit)
		if (e == errResponseLimit) != tc.exceeded {
			t.Fatal("wrong boundary")
		}
		if !tc.exceeded && string(b) != tc.body {
			t.Fatal("content changed")
		}
	}
}
func TestResponseLimitFailsClosed(t *testing.T) {
	root := gitFixture(t)
	fixtureServer(t, root, "large_valid")
	ctx := context.WithValue(context.Background(), unlimitedHistoryContext{}, true)
	ctx = context.WithValue(ctx, historyFetchModeContext{}, "paged500")
	ctx = context.WithValue(ctx, responseLimitContext{}, 1024)
	h := readHistory(ctx, root, []string{root}, map[string]bool{"file.go": true}, 0, time.Now().Unix())
	if h.Status == "complete" || h.Error == nil || h.Error.Code != "response_size_limit" || !compare(Input{History: h}).Fallback {
		t.Fatal("limit promoted to complete")
	}
}
