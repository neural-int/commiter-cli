package main

import (
	"encoding/json"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"reflect"
	"testing"
)

func TestCanonicalIRIgnoresIncomingFilePresentation(t *testing.T) {
	f := contractFixtures()[2]
	c, restore, e := canonicalFixture(f)
	if e != nil {
		t.Fatal(e)
	}
	reversed := f
	reversed.Files = append([]contextinput.File(nil), f.Files...)
	for i, j := 0, len(reversed.Files)-1; i < j; i, j = i+1, j-1 {
		reversed.Files[i], reversed.Files[j] = reversed.Files[j], reversed.Files[i]
	}
	other, r, e := canonicalFixture(reversed)
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(c.Files, other.Files) || !reflect.DeepEqual(c.Graph.Edges, other.Graph.Edges) || !reflect.DeepEqual(restore, r) {
		t.Fatal("incoming order leaked into canonical input")
	}
	a, _ := json.Marshal([]any{callFacts(c), assertionFacts(c), declarationFacts(c)})
	b, _ := json.Marshal([]any{callFacts(other), assertionFacts(other), declarationFacts(other)})
	if string(a) != string(b) {
		t.Fatal("IR order leaked")
	}
}
func TestConstantEntitiesPreserveObservedValues(t *testing.T) {
	f := fixtures()[6]
	facts := declarationFacts(f)
	if len(facts) != 32 {
		t.Fatal("constant fact dropped")
	}
	for _, fact := range facts {
		expected := "2"
		if fact.Version == "after" {
			expected = "3"
		}
		if fact.Kind != "const" || fact.Value != expected || fact.Name == "" {
			t.Fatal("constant fact invented")
		}
	}
}
