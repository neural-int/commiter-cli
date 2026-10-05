package main

// Fixed before H8 cross12 qualification result is inspected. This adds a shared
// protocol transition across different entities and a separate health change.
// Expected membership is author-defined evaluation data, never model input.
func contractHoldout() fixture {
	cases := []changeCase{
		sourceCase("transport/header.go", "transport", "", "func ProtocolHeader() string { return \"X-Protocol: 1\" }", "func ProtocolHeader() string { return \"X-Protocol: 2\" }"),
		testCase("transport/header_test.go", "transport", "ProtocolHeader", "if ProtocolHeader() != \"X-Protocol: 1\" { t.Fatal(\"header\") }", "if ProtocolHeader() != \"X-Protocol: 2\" { t.Fatal(\"header\") }"),
		sourceCase("wire/envelope.go", "wire", "", "func ProtocolEnvelope(payload string) string { return \"v1:\" + payload }", "func ProtocolEnvelope(payload string) string { return \"v2:\" + payload }"),
		testCase("wire/envelope_test.go", "wire", "ProtocolEnvelope", "if ProtocolEnvelope(\"x\") != \"v1:x\" { t.Fatal(\"envelope\") }", "if ProtocolEnvelope(\"x\") != \"v2:x\" { t.Fatal(\"envelope\") }"),
		sourceCase("reader/version.go", "reader", "", "func ProtocolVersionAllowed(v int) bool { return v == 1 }", "func ProtocolVersionAllowed(v int) bool { return v == 2 }"),
		testCase("reader/version_test.go", "reader", "ProtocolVersionAllowed", "if !ProtocolVersionAllowed(1) { t.Fatal(\"version1\") }; if ProtocolVersionAllowed(2) { t.Fatal(\"version2\") }", "if ProtocolVersionAllowed(1) { t.Fatal(\"version1\") }; if !ProtocolVersionAllowed(2) { t.Fatal(\"version2\") }"),
		sourceCase("health/live.go", "health", "", "func Live() string { return \"live\" }", "func Live() string { return \"ready\" }"),
		testCase("health/live_test.go", "health", "Live", "if Live() != \"live\" { t.Fatal(\"health\") }", "if Live() != \"ready\" { t.Fatal(\"health\") }"),
	}
	return fixtureFromCases("holdout-protocol-and-health-8", cases, [][]int{{0, 1, 2, 3, 4, 5}, {6, 7}})
}
