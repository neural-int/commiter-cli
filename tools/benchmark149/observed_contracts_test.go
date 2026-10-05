package main

import "testing"

func TestObservedContractGroundingRejectsSpoofedOrMissingEvidence(t *testing.T) {
	f, _, e := canonicalFixture(fixtures()[6])
	if e != nil {
		t.Fatal(e)
	}
	facts := observedEvidence(f)
	makeAnswer := func() contractOutput {
		a := contractOutput{}
		for _, fact := range facts {
			a.Contracts = append(a.Contracts, observedContract{fact.Subject, "observed old value", "observed new value", []string{fact.File}, []string{fact.ID}})
		}
		return a
	}
	if g, e := validateObservedContracts(f, facts, makeAnswer()); e != nil || len(g) != 16 {
		t.Fatal("grounded complete partition rejected")
	}
	for _, tc := range []struct {
		name   string
		mutate func(*contractOutput)
	}{
		{"unknown evidence", func(a *contractOutput) { a.Contracts[0].Evidence = []string{"E999"} }},
		{"other member evidence", func(a *contractOutput) { a.Contracts[0].Evidence = []string{facts[1].ID} }},
		{"duplicate member", func(a *contractOutput) { a.Contracts[1].Members = a.Contracts[0].Members }},
		{"missing file", func(a *contractOutput) { a.Contracts = a.Contracts[1:] }},
		{"unobserved subject", func(a *contractOutput) { a.Contracts[0].Subject = "invented shared policy" }},
		{"unresolved", func(a *contractOutput) { a.Unresolved = true }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := makeAnswer()
			tc.mutate(&a)
			g, e := validateObservedContracts(f, facts, a)
			if e == nil || g != nil {
				t.Fatal("invalid evidence produced a partial partition")
			}
		})
	}
}

func TestGroundedContractStructureDoesNotProveSemanticCorrectness(t *testing.T) {
	f, _, e := canonicalFixture(fixtures()[6])
	if e != nil {
		t.Fatal(e)
	}
	facts := observedEvidence(f)
	a := contractOutput{}
	for _, fact := range facts {
		a.Contracts = append(a.Contracts, observedContract{fact.Subject, "old value", "new value", []string{fact.File}, []string{fact.ID}})
	}
	a.Contracts[0].Members = append(a.Contracts[0].Members, a.Contracts[1].Members...)
	a.Contracts[0].Evidence = append(a.Contracts[0].Evidence, a.Contracts[1].Evidence...)
	a.Contracts = append(a.Contracts[:1], a.Contracts[2:]...)
	groups, e := validateObservedContracts(f, facts, a)
	if e != nil {
		t.Fatal(e)
	}
	// Restore identities only to score the independent purposes, never as input.
	_, restore, _ := canonicalFixture(fixtures()[6])
	for i := range groups {
		for j := range groups[i] {
			groups[i][j] = restore[groups[i][j]]
		}
	}
	exact, fm, fs := quality(groups, fixtures()[6].Expected)
	if exact || fm != 1 || fs != 0 {
		t.Fatal("structural grounding was confused with semantic proof")
	}
}

func TestDraftDiagnosticsCannotAuthorizeIncompleteEvidence(t *testing.T) {
	f, _, _ := canonicalFixture(contractFixtures()[0])
	facts := observedEvidence(f)
	contracts := []observedContract{}
	for i := 0; i < len(f.Files); i += 2 {
		members := []string{f.Files[i].ID, f.Files[i+1].ID}
		var fact contractEvidence
		for _, e := range facts {
			if e.File == members[0] {
				fact = e
				break
			}
		}
		contracts = append(contracts, observedContract{fact.Subject, "old contract", "new contract", members, []string{fact.ID}})
	}
	if groups, e := draftContractPartition(f, contracts); e != nil || len(groups) != 2 {
		t.Fatal("diagnostic lost valid selected partition")
	}
	if groups, e := validateObservedContracts(f, facts, contractOutput{Contracts: contracts}); e == nil || groups != nil {
		t.Fatal("draft diagnostic bypassed evidence gate")
	}
}
