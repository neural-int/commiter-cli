package main

import (
	"fmt"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/relation"
)

type fixture struct {
	Name     string
	Files    []contextinput.File
	Graph    relation.CandidateGraph
	Expected [][]string
}

// Expectations are kept out of the model payload. These are fixed synthetic
// changed-purpose fixtures, not production accuracy samples.
func fixtures() []fixture {
	out := []fixture{}
	for _, spec := range []struct {
		name         string
		n, groupSize int
		edges        bool
		cross        bool
	}{
		{"baseline-4", 4, 2, true, false},
		{"same-directory-independent-5", 5, 1, false, false},
		{"implementation-tests-8", 8, 4, true, false},
		{"cross-directory-single-intent-9", 9, 9, true, true},
		{"multiple-intents-boundary-12", 12, 6, true, true},
		{"missing-edges-single-intent-16", 16, 16, false, true},
		{"weak-edges-independent-16", 16, 1, false, false},
	} {
		f := fixture{Name: spec.name}
		changes := []gitstate.Change{}
		rels := relation.Result{}
		groups := map[int][]string{}
		topics := []string{"RetryLimit", "RequestTimeout", "CacheCapacity", "PageSize", "UploadLimit", "BatchSize", "QueueCapacity", "ConnectionLimit", "RetentionDays", "LogSize", "DownloadLimit", "SearchLimit", "WorkerCount", "BufferSize", "SessionLimit", "HistorySize"}
		for i := 0; i < spec.n; i++ {
			g := i / spec.groupSize
			id := fmt.Sprintf("F%03d", i+1)
			dir := "settings"
			if spec.cross {
				dir = fmt.Sprintf("component%02d", i)
			}
			path := fmt.Sprintf("%s/setting%02d.go", dir, i)
			test := i%2 == 1 && spec.groupSize > 1
			if test {
				path = fmt.Sprintf("%s/setting%02d_test.go", dir, i-1)
			}
			diff := fmt.Sprintf("@@ -1,2 +1,2 @@\n package settings\n-const %s = 2\n+const %s = 3\n", topics[g], topics[g])
			if test {
				diff = fmt.Sprintf("@@ -1,3 +1,3 @@\n func Test%s(t *testing.T) {\n- if %s != 2 { t.Fatal(\"limit\") }\n+ if %s != 3 { t.Fatal(\"limit\") }\n }\n", topics[g], topics[g], topics[g])
			}
			f.Files = append(f.Files, contextinput.File{ID: id, NewPath: &path, Status: "modified", Language: "go", RawDiff: diff})
			changes = append(changes, gitstate.Change{ID: id, NewPath: &path, Status: "modified"})
			groups[g] = append(groups[g], id)
			if spec.edges && i%spec.groupSize > 0 {
				rels.Relations = append(rels.Relations, relation.Relation{SourceID: fmt.Sprintf("F%03d", i), TargetID: id, Kind: relation.ChangedIdentifier, Class: relation.Soft, Reason: "controlled_changed_identifier_chain", Evidence: relation.Evidence{Type: "identifier", Value: topics[g]}})
			}
		}
		// Controlled graph inputs expose missing extraction edges. Path hints have
		// no connectivity even when all files share a directory.
		if !spec.edges {
			ids := []string{}
			for _, c := range changes {
				ids = append(ids, c.ID)
			}
			rels.Hints = []relation.Hint{{Kind: relation.PathProximity, FileIDs: ids, Evidence: relation.Evidence{Type: "directory", Value: "settings"}}}
		}
		var err error
		f.Graph, err = relation.BuildGraph(changes, rels)
		if err != nil {
			panic(err)
		}
		for g := 0; g < len(groups); g++ {
			f.Expected = append(f.Expected, groups[g])
		}
		out = append(out, f)
	}
	return out
}
