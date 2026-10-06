package main

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"path"
	"sort"
	"strings"
	"time"
)

type observerMetrics struct {
	Probes  int     `json:"probes"`
	Samples int     `json:"samples"`
	Unknown int     `json:"unknown_samples"`
	Wall    float64 `json:"wall_seconds"`
}
type snapshotValue struct {
	CalleeAfter bool `json:"callee_after"`
	CallerAfter bool `json:"caller_after"`
	Known       bool `json:"known"`
	Value       any  `json:"value,omitempty"`
}
type interactionFact struct {
	Callee    string          `json:"callee_file_id"`
	Caller    string          `json:"caller_file_id"`
	Function  string          `json:"observed_function"`
	Arguments []string        `json:"observed_arguments"`
	Snapshots []snapshotValue `json:"snapshots"`
	Contrast  string          `json:"effect_at_observed_input,omitempty"`
}

// Observed inputs come from test syntax, never expected values or gold groups.
// Zero/unknown probes do not remove any selected file from global assignment.
func interactionFacts(ctx context.Context, f fixture) ([]interactionFact, observerMetrics, error) {
	start := time.Now()
	m := observerMetrics{}
	finish := func(out []interactionFact, err error) ([]interactionFact, observerMetrics, error) {
		m.Wall = time.Since(start).Seconds()
		return out, m, err
	}
	paths := map[string]string{}
	for _, file := range f.Files {
		if file.NewPath != nil {
			paths[file.ID] = *file.NewPath
		}
	}
	edges := map[string]callFact{}
	for _, c := range callFacts(f) {
		if paths[c.Caller] != "" && !strings.HasSuffix(paths[c.Caller], "_test.go") {
			edges[c.Callee+"/"+c.Caller] = c
		}
	}
	keys := []string{}
	for k := range edges {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	assertions := assertionFacts(f)
	out := []interactionFact{}
	for _, key := range keys {
		c := edges[key]
		probes := map[string]assertionFact{}
		for _, a := range assertions {
			if a.Callee == c.Caller {
				probes[a.Function+"("+strings.Join(a.Arguments, ",")+")"] = a
			}
		}
		probeKeys := []string{}
		for k := range probes {
			probeKeys = append(probeKeys, k)
		}
		sort.Strings(probeKeys)
		for _, pk := range probeKeys {
			if m.Probes >= 32 {
				return finish(nil, errors.New("observation_budget"))
			}
			m.Probes++
			a := probes[pk]
			row := interactionFact{Callee: c.Callee, Caller: c.Caller, Function: a.Function, Arguments: a.Arguments, Snapshots: []snapshotValue{}}
			for _, bits := range [][2]bool{{false, false}, {true, false}, {false, true}, {true, true}} {
				if ctx.Err() != nil {
					return finish(nil, errors.New("observation_timeout"))
				}
				m.Samples++
				o := newPureObserver(f, map[string]bool{c.Callee: bits[0], c.Caller: bits[1]})
				args := []pureValue{}
				known := true
				for _, arg := range a.Arguments {
					e, err := parser.ParseExpr(arg)
					if err != nil {
						known = false
						break
					}
					if _, ok := e.(*ast.BasicLit); !ok {
						known = false
						break
					}
					v, err := o.expr(e, nil, pureFunction{}, 0)
					if err != nil {
						known = false
						break
					}
					args = append(args, v)
				}
				v := pureValue{}
				if known {
					var err error
					v, err = o.call(path.Dir(paths[c.Caller])+"/"+a.Function, args, 0)
					known = err == nil
				}
				s := snapshotValue{CalleeAfter: bits[0], CallerAfter: bits[1], Known: known}
				if known {
					s.Value = v.observed()
				} else {
					m.Unknown++
				}
				row.Snapshots = append(row.Snapshots, s)
			}
			out = append(out, row)
		}
	}
	return finish(out, nil)
}
