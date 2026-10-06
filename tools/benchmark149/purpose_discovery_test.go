package main

import "testing"

func TestPurposeDiscoveryEvidenceGates(t *testing.T) {
	no := false
	facts := []contractEvidence{{ID: "E001"}, {ID: "E002"}}
	valid := func() purposeDiscovery {
		return purposeDiscovery{[]provisionalPurpose{{"change A", []string{"E001"}}, {"change B", []string{"E001", "E002"}}}, &no}
	}
	if err := validatePurposeDiscovery(valid(), facts); err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*purposeDiscovery){
		"unknown":             func(d *purposeDiscovery) { d.Purposes[0].Evidence = []string{"E999"} },
		"duplicate reference": func(d *purposeDiscovery) { d.Purposes[0].Evidence = []string{"E001", "E001"} },
		"duplicate purpose":   func(d *purposeDiscovery) { d.Purposes[1].Purpose = "change A" },
		"missing status":      func(d *purposeDiscovery) { d.Unresolved = nil },
		"unresolved":          func(d *purposeDiscovery) { yes := true; d.Unresolved = &yes },
		"empty":               func(d *purposeDiscovery) { d.Purposes = nil },
		"empty label":         func(d *purposeDiscovery) { d.Purposes[0].Purpose = " " },
		"empty evidence":      func(d *purposeDiscovery) { d.Purposes[0].Evidence = nil },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			d := valid()
			mutate(&d)
			if validatePurposeDiscovery(d, facts) == nil {
				t.Fatal("accepted invalid discovery")
			}
		})
	}
}
