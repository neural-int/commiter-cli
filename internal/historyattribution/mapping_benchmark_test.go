package historyattribution

import (
	"fmt"
	"testing"
)

// Measure the review's accumulation risk without a model or real history.
func BenchmarkSplitMergeUnrelatedHistory(b *testing.B) {
	for _, count := range []int{100, 1000, 5000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			in := Input{Files: []GitFile{{Path: "current.go", Kind: "update", Diff: "@@ -0,0 +1 @@\n+target\n"}}, History: History{Status: "complete"}}
			// Both historical paths are also selected, as required by Collect.
			// Their mixed add/delete blocks do not enter this fragment search.
			in.Files = append(in.Files, GitFile{Path: "old0.go", Kind: "update", Diff: update("base", "changed")}, GitFile{Path: "old1.go", Kind: "update", Diff: update("base", "changed")})
			for i := 0; i < count; i++ {
				o := op(fmt.Sprint(i), fmt.Sprintf("@@ -0,0 +1 @@\n+unrelated%d\n", i))
				o.Path = fmt.Sprintf("old%d.go", i%2)
				in.History.Operations = append(in.History.Operations, o)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				out := Comparison{Results: []Attribution{{Path: "current.go", Block: 0, Status: "unmatched"}}}
				markSplitMergeEvidence(in, &out)
				if out.Results[0].Status != "unmatched" {
					b.Fatal("unrelated history became a candidate")
				}
			}
		})
	}
}
