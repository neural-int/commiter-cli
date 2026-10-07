package main

import (
	"encoding/json"
	"os"
	"testing"
)

// Optional synthetic fixture export; gold is isolated from extraction input.
func TestExport151(t *testing.T) {
	path := os.Getenv("BENCHMARK151_EXPORT")
	if path == "" {
		t.Skip("explicit export path required")
	}
	type file struct {
		ID     string
		Path   string
		Before string
		After  string
	}
	type record struct {
		Name  string
		Files []file
		Gold  [][]string
	}
	fs := append(fixtures(), contractFixtures()...)
	fs = append(fs, holdouts()...)
	fs = append(fs, holdout16(), sharedCalleeGuardrail(), wireHoldout())
	out := []record{}
	for _, f := range fs {
		r := record{Name: f.Name, Gold: f.Expected}
		for _, x := range f.Files {
			r.Files = append(r.Files, file{x.ID, *x.NewPath, versionSource(x.RawDiff, false), versionSource(x.RawDiff, true)})
		}
		out = append(out, r)
	}
	data, e := json.Marshal(out)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(path, data, 0600); e != nil {
		t.Fatal(e)
	}
}
