package main

// A shared selected callee changes whitespace handling; its callers change
// unrelated prefixes and delimiters on whitespace-free inputs. Call edges are
// real, but their presence does not prove one changed purpose.
func sharedCalleeGuardrail() fixture {
	a := sourceCase("utility/normalize.go", "utility", "import \"strings\"", "func Normalize(s string) string {return strings.ToLower(s)}", "func Normalize(s string) string {return strings.ToLower(strings.TrimSpace(s))}")
	at := testCase("utility/normalize_test.go", "utility", "NormalizedWhitespace", "if Normalize(\" A \")!=\" a \" {t.Fatal(\"untrimmed label\")}", "if Normalize(\" A \")!=\"a\" {t.Fatal(\"trimmed label\")}")
	b := sourceCase("trace/label.go", "trace", "import \"fixture/utility\"", "func Label(s string) string {return utility.Normalize(s)}", "func Label(s string) string {return \"trace:\"+utility.Normalize(s)}")
	bt := testCase("trace/label_test.go", "trace", "TracePrefix", "if Label(\"A\")!=\"a\" {t.Fatal(\"trace\")}", "if Label(\"A\")!=\"trace:a\" {t.Fatal(\"trace prefix\")}")
	c := sourceCase("render/label.go", "render", "import \"fixture/utility\"", "func Label(s string) string {return utility.Normalize(s)}", "func Label(s string) string {return \"[\"+utility.Normalize(s)+\"]\"}")
	ct := testCase("render/label_test.go", "render", "LabelDelimiters", "if Label(\"A\")!=\"a\" {t.Fatal(\"rendered label\")}", "if Label(\"A\")!=\"[a]\" {t.Fatal(\"rendered delimiters\")}")
	return fixtureFromCases("shared-callee-independent-6", []changeCase{a, at, b, bt, c, ct}, [][]int{{0, 1}, {2, 3}, {4, 5}})
}
