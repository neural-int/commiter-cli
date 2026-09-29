package main

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestIssue142PairLocalInputScopeAndOrder(t *testing.T) {
	items, err := issue142LoadHoldoutFixtures(filepath.Join("..", "..", issue142CappedFixturePath))
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]fixture{}
	for _, item := range items {
		byName[item.name] = item
	}
	for name, expected := range map[string]string{"new_paraphrase": "same", "new_stem_doc_diverged": "different"} {
		item := byName[name]
		gold, err := issue142PairLocalGold(item)
		if err != nil || gold != expected {
			t.Fatalf("%s gold=%s err=%v", name, gold, err)
		}
		for _, order := range issue142PairLocalOrders {
			_, prompt, schema, err := issue142PairLocalInput(item, order)
			if err != nil {
				t.Fatal(err)
			}
			var decoded struct {
				Task  string                  `json:"task"`
				Files []issue142PairLocalFile `json:"files"`
			}
			if err := json.Unmarshal(prompt, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.Task != issue142PairLocalTask || len(decoded.Files) != 2 || decoded.Files[0].ID != order.FileIDs[0] || decoded.Files[1].ID != order.FileIDs[1] {
				t.Fatalf("%s: pair-local prompt scope/order changed: %+v", name, decoded)
			}
			var shape struct {
				Properties struct {
					Decision struct {
						Enum []string `json:"enum"`
					} `json:"decision"`
				} `json:"properties"`
			}
			if err := json.Unmarshal(schema, &shape); err != nil {
				t.Fatal(err)
			}
			if len(shape.Properties.Decision.Enum) != 2 || shape.Properties.Decision.Enum[0] != order.Enum[0] || shape.Properties.Decision.Enum[1] != order.Enum[1] {
				t.Fatalf("%s: enum order changed", name)
			}
		}
	}
}
