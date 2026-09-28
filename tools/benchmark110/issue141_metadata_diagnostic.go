package main

type issue141MetadataDiagnostic struct {
	ParseFailure    string `json:"parse_failure,omitempty"`
	ExpectedGroups  int    `json:"expected_groups"`
	ObservedItems   int    `json:"observed_items"`
	MissingGroups   int    `json:"missing_groups"`
	UnknownGroups   int    `json:"unknown_groups"`
	DuplicateGroups int    `json:"duplicate_groups"`
	MissingBreaking int    `json:"missing_breaking"`
}

// This returns counts only. Prompt text, response text, and generated group
// labels must not be stored in benchmark artifacts.
func issue141DiagnoseMetadata(data []byte, groups []issue141Group) issue141MetadataDiagnostic {
	diagnostic := issue141MetadataDiagnostic{ExpectedGroups: len(groups)}
	var output issue141MetadataOutput
	if failure := issue141Decode(data, &output); failure != "" {
		diagnostic.ParseFailure = failure
		return diagnostic
	}
	diagnostic.ObservedItems = len(output.Metadata)
	expected := make(map[string]bool, len(groups))
	seen := make(map[string]bool, len(groups))
	for _, group := range groups {
		expected[group.GroupID] = true
	}
	for _, item := range output.Metadata {
		if !expected[item.GroupID] {
			diagnostic.UnknownGroups++
		} else if seen[item.GroupID] {
			diagnostic.DuplicateGroups++
		}
		seen[item.GroupID] = true
		if item.Breaking == nil {
			diagnostic.MissingBreaking++
		}
	}
	for id := range expected {
		if !seen[id] {
			diagnostic.MissingGroups++
		}
	}
	return diagnostic
}
