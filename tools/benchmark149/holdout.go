package main

import "fmt"

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

// Larger independent evaluation combines five earlier declared behavior
// contracts with three new contracts; gold remains absent from model payload.
func holdout16() fixture {
	a := sourceCase("engine/divide.go", "engine", "", "func Divide(n,d int) int { return n/d }", "func Divide(n,d int) int { if d==0 {return 0}; return n/d }")
	at := testCase("engine/divide_test.go", "engine", "SafeDivision", `if Divide(6,2)!=3 {t.Fatal("division")}`, `if Divide(6,2)!=3 || Divide(6,0)!=0 {t.Fatal("safe division")}`)
	b := sourceCase("engine/clamp.go", "engine", "", "func Clamp(n int) int { if n<0 {return 0};return n }", "func Clamp(n int) int { if n<0 {return 0};if n>100 {return 100};return n }")
	bt := testCase("engine/clamp_test.go", "engine", "UpperClamp", `if Clamp(101)!=101 {t.Fatal("unbounded")}`, `if Clamp(101)!=100 {t.Fatal("upper bound")}`)
	c := sourceCase("engine/first.go", "engine", "import \"unicode/utf8\"", "func First(s string) rune { _=utf8.RuneError; if len(s)==0{return 0};return rune(s[0]) }", "func First(s string) rune { if len(s)==0{return 0};r,_:=utf8.DecodeRuneInString(s);return r }")
	ct := testCase("engine/first_test.go", "engine", "FirstRune", "if First(\"あ\")!=227 {t.Fatal(\"first byte\")}", "if First(\"あ\")!='あ' {t.Fatal(\"first character\")}")
	fresh := fixtureFromCases("fresh", []changeCase{a, at, b, bt, c, ct}, [][]int{{0, 1}, {2, 3}, {4, 5}})
	out := fixture{Name: "holdout-independent-16"}
	parts := []fixture{contractFixtures()[0], holdouts()[0], fresh}
	for _, f := range parts {
		offset := len(out.Files)
		for i, file := range f.Files {
			file.ID = fmt.Sprintf("F%03d", offset+i+1)
			out.Files = append(out.Files, file)
		}
		for _, group := range f.Expected {
			ids := []string{}
			for _, id := range group {
				var n int
				fmt.Sscanf(id, "F%d", &n)
				ids = append(ids, fmt.Sprintf("F%03d", offset+n))
			}
			out.Expected = append(out.Expected, ids)
		}
	}
	return out
}
