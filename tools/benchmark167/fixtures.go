package main

import (
	"fmt"
	"strings"
)

type fixtureFile struct {
	Path   string `json:"path"`
	Before string `json:"before"`
	After  string `json:"after"`
	Intent int    `json:"intent"`
}
type fixture struct {
	Name  string        `json:"name"`
	Files []fixtureFile `json:"files"`
}

// Gold purposes are fixed before inference; these fixtures were not used by
// the earlier research or by the metadata regression tests.
func fixtures() []fixture {
	result := []fixture{}
	for _, spec := range []struct {
		name    string
		count   int
		pattern string
	}{
		{"one-independent", 1, "independent"},
		{"source-test-pair", 2, "pairs"},
		{"cross-directory-purpose", 3, "cross"},
		{"boundary-base", 4, "independent"},
		{"boundary-plus-consumer", 5, "boundary"},
		{"four-source-test-pairs", 8, "pairs"},
		{"cross-directory-scale", 9, "cross"},
		{"independent-scale", 16, "independent"},
	} {
		f := fixture{Name: spec.name}
		for i := 0; i < spec.count; i++ {
			intent := i
			dir := "settings"
			isTest := false
			switch spec.pattern {
			case "pairs":
				intent = i / 2
				dir = fmt.Sprintf("module%d", intent)
				isTest = i%2 == 1
			case "cross":
				intent = 0
				dir = fmt.Sprintf("component%d", i)
			case "boundary":
				if i == 4 {
					intent = 0
					isTest = true
				}
			}
			name := fmt.Sprintf("WithinLimit%d", intent)
			path := fmt.Sprintf("%s/limit%d.go", dir, intent)
			before := fmt.Sprintf("package settings\n\nfunc %s(value int) bool { return value < %d }\n", name, 10+intent)
			after := strings.Replace(before, "value < ", "value <= ", 1)
			if isTest {
				path = fmt.Sprintf("%s/limit%d_test.go", dir, intent)
				before = fmt.Sprintf("package settings\n\nimport \"testing\"\n\nfunc Test%sBoundary(t *testing.T) {\n if %s(%d) { t.Fatal(\"boundary included\") }\n}\n", name, name, 10+intent)
				after = fmt.Sprintf("package settings\n\nimport \"testing\"\n\nfunc Test%sBoundary(t *testing.T) {\n if !%s(%d) { t.Fatal(\"boundary excluded\") }\n}\n", name, name, 10+intent)
			}
			f.Files = append(f.Files, fixtureFile{Path: path, Before: before, After: after, Intent: intent})
		}
		result = append(result, f)
	}
	return result
}
