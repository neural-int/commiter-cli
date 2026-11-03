package llm

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLiteralNewlineDisplayPreservesTransportAndCodeEscapes(t *testing.T) {
	original := `{"raw_diff":"@@ -1 +1 @@\n-pattern=\"\\n\"\n+pattern=\"\\t\"\n","text":"日本語\n次"}`
	display, err := LiteralNewlineDisplay([]byte(original))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(display, "@@ -1 +1 @@\n-") || !strings.Contains(display, `\\n`) {
		t.Fatal("display lost newline/code escape")
	}
	transport, _ := json.Marshal(Message{Role: "user", Content: display})
	if !json.Valid(transport) {
		t.Fatal("invalid IPC")
	}
	var decoded Message
	json.Unmarshal(transport, &decoded)
	if decoded.Content != display {
		t.Fatal("transport changed display")
	}
	if _, err = LiteralNewlineDisplay([]byte("invalid")); err == nil {
		t.Fatal("accepted invalid canonical JSON")
	}
}
