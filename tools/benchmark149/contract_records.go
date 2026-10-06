package main

// Records associate only uniquely resolved syntax observations. These references
// are evidence presentation, never provisional or final grouping constraints.
type contractRecord struct {
	contractEvidence
	Calls []callFact      `json:"observed_consumers"`
	Tests []assertionFact `json:"observed_test_assertions"`
}

func contractRecords(f fixture, evidence []contractEvidence) ([]contractRecord, []assertionFact, []callFact) {
	calls, assertions := callFacts(f), assertionFacts(f)
	records := make([]contractRecord, 0, len(evidence))
	attached := make([]bool, len(assertions))
	attachedCalls := make([]bool, len(calls))
	for _, e := range evidence {
		r := contractRecord{contractEvidence: e, Calls: []callFact{}, Tests: []assertionFact{}}
		for i, c := range calls {
			if c.Callee == e.File && c.Symbol == e.Subject {
				r.Calls = append(r.Calls, c)
				attachedCalls[i] = true
			}
		}
		for i, a := range assertions {
			if a.Callee != "" && a.Callee == e.File && a.Function == e.Subject {
				r.Tests = append(r.Tests, a)
				attached[i] = true
			}
		}
		records = append(records, r)
	}
	unassociated := []assertionFact{}
	for i, a := range assertions {
		if !attached[i] {
			unassociated = append(unassociated, a)
		}
	}
	unassociatedCalls := []callFact{}
	for i, c := range calls {
		if !attachedCalls[i] {
			unassociatedCalls = append(unassociatedCalls, c)
		}
	}
	return records, unassociated, unassociatedCalls
}
