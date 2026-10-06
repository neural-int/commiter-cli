package main

import (
	"encoding/json"
	"go/parser"
	"os"
	"path"
	"testing"
)

func TestPureObserverMatchesRecordedGoExecutions(t *testing.T) {
	data, err := os.ReadFile("../../docs/benchmarks/issue-149/iteration-17-runtime-interactions.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []interactionObservation
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	fixtures := map[string]fixture{}
	for _, f := range []fixture{contractFixtures()[2], sharedCalleeGuardrail()} {
		fixtures[f.Name] = f
	}
	matched := 0
	for _, r := range rows {
		f, ok := fixtures[r.Fixture]
		if !ok {
			t.Fatal("unknown recorded fixture")
		}
		o := newPureObserver(f, map[string]bool{r.Callee: r.CalleeAfter, r.Caller: r.CallerAfter})
		args := []pureValue{}
		for _, arg := range r.Arguments {
			e, err := parser.ParseExpr(arg)
			if err != nil {
				t.Fatal(err)
			}
			v, err := o.expr(e, nil, pureFunction{}, 0)
			if err != nil {
				t.Fatal(err)
			}
			args = append(args, v)
		}
		pkg := ""
		for _, file := range f.Files {
			if file.ID == r.Caller {
				pkg = path.Dir(*file.NewPath)
			}
		}
		v, err := o.call(pkg+"/"+r.Function, args, 0)
		if err != nil {
			t.Fatal(err)
		}
		got, err := json.Marshal(v.observed())
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(r.Value) {
			t.Fatalf("%s %s: observer=%s runtime=%s", r.Fixture, r.Function, got, r.Value)
		}
		matched++
	}
	if matched != 24 {
		t.Fatal("incomplete runtime oracle coverage")
	}
	t.Logf("pure_observer matched_runtime_observations=%d", matched)
}

func TestPureObserverRejectsUnknownAndBoundsEvaluation(t *testing.T) {
	for _, tc := range []struct{ name, imports, body string }{
		{"mutation", "", "func Probe() int { x:=1; x++; return x }"},
		{"external function", "import \"os\"", "func Probe() string { return os.Getenv(\"BENCHMARK149_NOT_READ\") }"},
		{"recursion", "", "func Probe() int {return Probe()}"},
		{"integer range", "", "func Probe() int64 {return 67108865}"},
		{"unresolved symbol", "", "func Probe() int {return Missing()}"},
		{"exact floating constant semantics", "", "func Probe() bool {return 0.1+0.2==0.3}"},
		{"implicit fractional return", "", "func Probe() int {return 1.5}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := sourceCase("core/probe.go", "core", tc.imports, tc.body, tc.body)
			f := fixtureFromCases("pure-unknown", []changeCase{c}, nil)
			o := newPureObserver(f, nil)
			if _, err := o.call("core/Probe", nil, 0); err == nil {
				t.Fatal("unsupported evaluation accepted")
			}
		})
	}
	if _, err := (&pureObserver{steps: 0}).call("missing", nil, 0); err == nil {
		t.Fatal("empty budget accepted")
	}
}
