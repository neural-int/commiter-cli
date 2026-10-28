package main

import (
	"fmt"
	"strings"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/relation"
	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

type changeCase struct{ Path, Before, After string }

func sourceCase(path, pkg, imports, before, after string) changeCase {
	prefix := "package " + pkg + "\n" + imports + "\n"
	return changeCase{path, prefix + before + "\n", prefix + after + "\n"}
}
func testCase(path, pkg, name, before, after string) changeCase {
	prefix := "package " + pkg + "\nimport \"testing\"\nfunc Test" + name + "(t *testing.T) {\n"
	return changeCase{path, prefix + before + "\n}\n", prefix + after + "\n}\n"}
}
func fixtureFromCases(name string, cases []changeCase, groups [][]int) fixture {
	f := fixture{Name: name}
	changes := []gitstate.Change{}
	inputs := []relation.File{}
	for i, c := range cases {
		id := fmt.Sprintf("F%03d", i+1)
		path := c.Path
		content := []byte(c.After)
		changed := gitstate.Change{ID: id, Status: "modified", NewPath: &path, Language: "go", WorktreeKind: "file"}
		analysis := syntax.Analyze(syntax.Input{Language: "go", Content: content, Hunks: []syntax.Hunk{{StartLine: 1, EndLine: strings.Count(c.After, "\n") + 1}}})
		if analysis.Mode != syntax.ModeStructural {
			panic("contract fixture must parse")
		}
		diff := fmt.Sprintf("@@ -1,%d +1,%d @@\n", strings.Count(c.Before, "\n"), strings.Count(c.After, "\n"))
		for _, line := range strings.Split(strings.TrimSuffix(c.Before, "\n"), "\n") {
			diff += "-" + line + "\n"
		}
		for _, line := range strings.Split(strings.TrimSuffix(c.After, "\n"), "\n") {
			diff += "+" + line + "\n"
		}
		f.Files = append(f.Files, contextinput.File{ID: id, NewPath: &path, Status: "modified", Language: "go", RawDiff: diff, Mode: syntax.ModeRawDiff})
		changes = append(changes, changed)
		inputs = append(inputs, relation.File{Change: changed, Content: content, Evidence: analysis.Evidence})
	}
	extracted, err := relation.Extract(inputs)
	if err != nil {
		panic(err)
	}
	f.Graph, err = relation.BuildGraph(changes, extracted)
	if err != nil {
		panic(err)
	}
	for _, indices := range groups {
		ids := []string{}
		for _, i := range indices {
			ids = append(ids, fmt.Sprintf("F%03d", i+1))
		}
		f.Expected = append(f.Expected, ids)
	}
	return f
}

// These fixtures are declared before their first inference. Unlike the initial
// constant-change probes, each independent group has a distinct observable bug
// and an accompanying regression contract. The production extractor builds the
// graph from parsed after-content; no edges are supplied from expected groups.
func contractFixtures() []fixture {
	expiry := sourceCase("core/expiry.go", "core", "", "func Expired(now, deadline int64) bool { return now > deadline }", "func Expired(now, deadline int64) bool { return now >= deadline }")
	expiryTest := testCase("core/expiry_test.go", "core", "ExpiryBoundary", "if Expired(10, 10) { t.Fatal(\"deadline equality\") }", "if !Expired(10, 10) { t.Fatal(\"deadline must expire\") }")
	cents := sourceCase("core/cents.go", "core", "import \"math\"", "func Cents(total float64) int { return int(math.Floor(total * 100)) }", "func Cents(total float64) int { return int(math.Round(total * 100)) }")
	centsTest := testCase("core/cents_test.go", "core", "CentRounding", "if Cents(1.999) != 199 { t.Fatal(\"amount\") }", "if Cents(1.999) != 200 { t.Fatal(\"amount must round\") }")
	capacity := sourceCase("core/capacity.go", "core", "", "func Capacity(limit int) int { if limit < 0 { return 1 }; return limit }", "func Capacity(limit int) int { if limit <= 0 { return 1 }; return limit }")
	capacityTest := testCase("core/capacity_test.go", "core", "ZeroCapacity", "if Capacity(0) != 0 { t.Fatal(\"capacity\") }", "if Capacity(0) != 1 { t.Fatal(\"capacity must be positive\") }")
	out := []fixture{
		fixtureFromCases("contract-baseline-4", []changeCase{expiry, expiryTest, cents, centsTest}, [][]int{{0, 1}, {2, 3}}),
		fixtureFromCases("contract-independent-6", []changeCase{expiry, expiryTest, cents, centsTest, capacity, capacityTest}, [][]int{{0, 1}, {2, 3}, {4, 5}}),
	}
	auth := []changeCase{
		sourceCase("auth/expiry.go", "auth", "", "func Expired(now, deadline int64) bool { return now > deadline }", "func Expired(now, deadline int64) bool { return now >= deadline }"),
		testCase("auth/expiry_test.go", "auth", "ExpiryBoundary", "if Expired(10, 10) { t.Fatal(\"deadline equality\") }", "if !Expired(10, 10) { t.Fatal(\"deadline must expire\") }"),
		sourceCase("gateway/auth.go", "gateway", "import \"fixture/auth\"", "func Authorized(now, deadline int64) bool { return !auth.Expired(now, deadline) || now == deadline }", "func Authorized(now, deadline int64) bool { return !auth.Expired(now, deadline) }"),
		testCase("gateway/auth_test.go", "gateway", "ExpiredAuthorization", "if !Authorized(10, 10) { t.Fatal(\"authorization\") }", "if Authorized(10, 10) { t.Fatal(\"expired authorization must fail\") }"),
		sourceCase("storage/session.go", "storage", "import \"fixture/auth\"", "func Restore(now, deadline int64) bool { return !auth.Expired(now, deadline) || deadline == now }", "func Restore(now, deadline int64) bool { return !auth.Expired(now, deadline) }"),
		testCase("storage/session_test.go", "storage", "ExpiredRestore", "if !Restore(10, 10) { t.Fatal(\"restore\") }", "if Restore(10, 10) { t.Fatal(\"expired session must not restore\") }"),
	}
	money := []changeCase{
		sourceCase("billing/cents.go", "billing", "import \"math\"", "func Cents(total float64) int { return int(math.Floor(total * 100)) }", "func Cents(total float64) int { return int(math.Round(total * 100)) }"),
		testCase("billing/cents_test.go", "billing", "CentRounding", "if Cents(1.999) != 199 { t.Fatal(\"amount\") }", "if Cents(1.999) != 200 { t.Fatal(\"amount must round\") }"),
		sourceCase("receipts/totals.go", "receipts", "import \"fixture/billing\"", "func FormattedCents(total float64) int { return billing.Cents(total - 0.005) }", "func FormattedCents(total float64) int { return billing.Cents(total) }"),
		testCase("receipts/totals_test.go", "receipts", "RoundedReceipt", "if FormattedCents(1.999) != 199 { t.Fatal(\"receipt\") }", "if FormattedCents(1.999) != 200 { t.Fatal(\"receipt must round\") }"),
		sourceCase("orders/amount.go", "orders", "import \"fixture/billing\"", "func PersistedCents(total float64) int { return billing.Cents(total - 0.005) }", "func PersistedCents(total float64) int { return billing.Cents(total) }"),
		testCase("orders/amount_test.go", "orders", "RoundedOrder", "if PersistedCents(1.999) != 199 { t.Fatal(\"order\") }", "if PersistedCents(1.999) != 200 { t.Fatal(\"order must round\") }"),
	}
	out = append(out, fixtureFromCases("contract-cross-boundary-12", append(auth, money...), [][]int{{0, 1, 2, 3, 4, 5}, {6, 7, 8, 9, 10, 11}}))
	return out
}
