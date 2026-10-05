package main

import "testing"

func TestObservedRoundingAssertionsKeepLiteralExpectations(t *testing.T) {
	f := contractFixtures()[2]
	facts := assertionFacts(f)
	for _, version := range []string{"before", "after"} {
		for _, id := range []string{"F008", "F010", "F012"} {
			found := false
			for _, a := range facts {
				if a.File == id && a.Version == version {
					expected := "199"
					if version == "after" {
						expected = "200"
					}
					if a.Expected != expected || a.Comparison != "equal" || len(a.Arguments) != 1 || a.Arguments[0] != "1.999" || a.Callee == "" {
						t.Fatalf("unsupported assertion reinterpretation: %+v", a)
					}
					found = true
				}
			}
			if !found {
				t.Fatal("assertion omitted")
			}
		}
	}
}
