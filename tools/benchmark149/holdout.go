package main

// Declared before any Issue149 model result is inspected. Author-defined
// synthetic contracts; these are independent evaluation, not real-world gold.
func holdouts() []fixture {
	a := sourceCase("engine/remainder.go", "engine", "", "func Remainder(n, d int) int { return n % d }", "func Remainder(n, d int) int { r := n % d; if r < 0 { r += d }; return r }")
	at := testCase("engine/remainder_test.go", "engine", "PositiveRemainder", "if Remainder(-1, 3) != -1 { t.Fatal(\"signed remainder\") }", "if Remainder(-1, 3) != 2 { t.Fatal(\"positive remainder\") }")
	b := sourceCase("engine/label.go", "engine", "import \"strings\"", "func Label(s string) string { return strings.ToLower(s) }", "func Label(s string) string { return strings.ToLower(strings.TrimSpace(s)) }")
	bt := testCase("engine/label_test.go", "engine", "TrimmedLabel", "if Label(\" A \") != \" a \" { t.Fatal(\"label\") }", "if Label(\" A \") != \"a\" { t.Fatal(\"trimmed label\") }")
	c := sourceCase("engine/length.go", "engine", "import \"unicode/utf8\"", "func Length(s string) int { _ = utf8.RuneCountInString; return len(s) }", "func Length(s string) int { return utf8.RuneCountInString(s) }")
	ct := testCase("engine/length_test.go", "engine", "UnicodeLength", "if Length(\"あ\") != 3 { t.Fatal(\"byte length\") }", "if Length(\"あ\") != 1 { t.Fatal(\"character length\") }")
	return []fixture{fixtureFromCases("holdout-independent-6", []changeCase{a, at, b, bt, c, ct}, [][]int{{0, 1}, {2, 3}, {4, 5}})}
}
