package codexcontext

import (
	"context"
	"os"
	"testing"
	"time"
)

// Explicitly opt in: normal tests never open the developer's Codex history.
func TestLiveAppServerReadOnly(t *testing.T) {
	if os.Getenv("COMMITER_CODEX_LIVE_SMOKE") != "1" {
		t.Skip("opt-in local history read")
	}
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	p, err := start(ctx, root)
	if err != nil {
		t.Fatal("cannot start local Codex transport")
	}
	defer p.close()
	if p.initialize() != nil {
		t.Fatal("cannot initialize local Codex transport")
	}
	var threads page[thread]
	if p.call("thread/list", map[string]any{"limit": PageSize, "useStateDbOnly": true, "sortKey": "updated_at"}, &threads) != nil || threads.Data == nil {
		t.Fatal("cannot list local threads")
	}
	if len(threads.Data) == 0 {
		t.Skip("no local threads")
	}
	th := threads.Data[0]
	var turns page[turn]
	if p.call("thread/turns/list", map[string]any{"threadId": th.ID, "limit": PageSize, "itemsView": "notLoaded"}, &turns) != nil || turns.Data == nil {
		t.Fatal("cannot page local turns")
	}
	if len(turns.Data) > 0 {
		var items page[entry]
		if p.call("thread/items/list", map[string]any{"threadId": th.ID, "turnId": turns.Data[0].ID, "limit": PageSize}, &items) != nil || items.Data == nil {
			t.Fatal("cannot page local items")
		}
		t.Logf("read-only API smoke: threads=%d turns=%d items=%d bytes=%d elapsed=%s", len(threads.Data), len(turns.Data), len(items.Data), MaxReceiveBytes+1-p.reader.N, time.Since(started))
	}
}
