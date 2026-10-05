package main

import "testing"

func TestObservedAnchorRejectsCyclesAndInvalidIdentities(t *testing.T) {
	f, _, _ := canonicalFixture(contractFixtures()[0])
	facts := anchorEvidence(f)
	owner := map[string]string{}
	for _, fact := range facts {
		owner[fact.File] = fact.ID
	}
	valid := map[string]string{}
	for _, file := range f.Files {
		valid[file.ID] = owner[file.ID]
	}
	if _, e := validateAnchorAssignment(f, facts, valid); e != nil {
		t.Fatal(e)
	}
	copyMap := func() map[string]string {
		m := map[string]string{}
		for k, v := range valid {
			m[k] = v
		}
		return m
	}
	for _, tc := range []struct {
		name   string
		mutate func(map[string]string)
	}{
		{"cycle", func(m map[string]string) {
			m[f.Files[0].ID] = owner[f.Files[1].ID]
			m[f.Files[1].ID] = owner[f.Files[0].ID]
		}},
		{"unknown root", func(m map[string]string) { m[f.Files[0].ID] = "E999" }},
		{"unknown file", func(m map[string]string) { m["S999"] = "E001" }},
		{"missing file", func(m map[string]string) { delete(m, f.Files[0].ID) }},
		{"unresolved", func(m map[string]string) { m[f.Files[0].ID] = "unresolved" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := copyMap()
			tc.mutate(m)
			g, e := validateAnchorAssignment(f, facts, m)
			if e == nil || g != nil {
				t.Fatal("invalid roots produced partial grouping")
			}
		})
	}
}
func TestAnchorFallbackKeepsObservedPartialSnippet(t *testing.T) {
	f, _, _ := canonicalFixture(fixtures()[2])
	facts := anchorEvidence(f)
	covered := map[string]bool{}
	literalCount := 0
	for _, fact := range facts {
		covered[fact.File] = true
		if fact.Kind == "diff-literal" {
			literalCount++
			for _, file := range f.Files {
				if file.ID == fact.File && (fact.Before != versionSource(file.RawDiff, false) || fact.After != versionSource(file.RawDiff, true)) {
					t.Fatal("fallback invented code")
				}
			}
		}
	}
	if len(covered) != len(f.Files) || literalCount == 0 {
		t.Fatal("partial observed file dropped")
	}
}
