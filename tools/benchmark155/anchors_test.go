package main

import "testing"

func TestPredicateEvidencePreservesUTF8AndCompoundConditions(t *testing.T) {
	old := "package example\nimport \"testing\"\nfunc TestBoth(t *testing.T) { t.Log(\"日本語\"); if Left(2)!=3 || Right(2)!=4 { t.Fatal(\"bad\") } }\n"
	next := "package example\nimport \"testing\"\nfunc TestBoth(t *testing.T) { t.Log(\"日本語\"); if Left(2)!=4 || Right(2)!=4 { t.Fatal(\"message changed\") } }\n"
	result, err := Extract(Request{[]Snapshot{{"F001", "both_test.go", old, next}}})
	if err != nil {
		t.Fatal(err)
	}
	predicates, changed := 0, 0
	for _, e := range result.Evidence {
		src := old
		if e.Version == "after" {
			src = next
		}
		if src[e.Span[0]:e.Span[1]] != e.Text || sha(src) != e.SourceSHA {
			t.Fatal("invalid source evidence")
		}
		if e.Kind == "failure_predicate" {
			predicates++
			if e.Changed {
				changed++
			}
			if e.ConditionSpan[0] > e.Span[0] || e.ConditionSpan[1] < e.Span[1] {
				t.Fatal("lost compound condition")
			}
		}
	}
	if predicates != 4 || changed != 2 {
		t.Fatalf("predicates=%d changed=%d", predicates, changed)
	}
}
func TestUnavailableObservationIsExplicitAndInvalidIDsReject(t *testing.T) {
	result, err := Extract(Request{[]Snapshot{{"F1", "opaque.py", "x=1", "x=2"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Evidence) != 0 || len(result.Status) != 2 || result.Status[0].Status != "unsupported_language" {
		t.Fatal("fabricated observation")
	}
	_, err = Extract(Request{[]Snapshot{{ID: "F1"}, {ID: "F1"}}})
	if err == nil {
		t.Fatal("duplicate accepted")
	}
}

func TestContextFactsRetainTypeDeclarationsAndLocalCallNames(t *testing.T) {
	old := "package sample\ntype Result struct { N int }\nfunc Adapter() Result { return Core() }\nfunc Core() Result { return Result{N:1} }\n"
	next := "package sample\ntype Result struct { N int; Count int }\nfunc Adapter() Result { return Core() }\nfunc Core() Result { return Result{N:2} }\n"
	result, err := Extract(Request{[]Snapshot{{"F1", "value.go", old, next}}})
	if err != nil {
		t.Fatal(err)
	}
	changedTypes, adapterCalls := 0, 0
	for _, e := range result.Evidence {
		if e.Kind == "type_source" && e.Function == "Result" && e.Changed {
			changedTypes++
		}
		if e.Kind == "function_source" && e.Function == "Adapter" && len(e.Calls) == 1 && e.Calls[0] == "Core" && !e.Changed {
			adapterCalls++
		}
	}
	if changedTypes != 2 || adapterCalls != 2 {
		t.Fatalf("types=%d calls=%d", changedTypes, adapterCalls)
	}
}
