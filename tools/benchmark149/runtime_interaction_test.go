package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

type interactionObservation struct {
	Fixture     string          `json:"fixture"`
	Callee      string          `json:"callee_file_id"`
	Caller      string          `json:"caller_file_id"`
	Function    string          `json:"observed_function"`
	Arguments   []string        `json:"observed_arguments"`
	CalleeAfter bool            `json:"callee_after"`
	CallerAfter bool            `json:"caller_after"`
	Value       json.RawMessage `json:"observed_value"`
	Wall        float64         `json:"wall_seconds"`
}

// Executes only repository-defined synthetic fixtures, never collected user
// changes. Results are observations at fixed test inputs, not commit boundaries.
func TestSyntheticRuntimeInteractionProbe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()
	all := []interactionObservation{}
	for _, f := range []fixture{contractFixtures()[2], sharedCalleeGuardrail()} {
		byID := map[string]int{}
		for i, file := range f.Files {
			byID[file.ID] = i
		}
		unique := map[string]callFact{}
		for _, c := range callFacts(f) {
			if strings.HasSuffix(*f.Files[byID[c.Caller]].NewPath, "_test.go") {
				continue
			}
			unique[c.Callee+"/"+c.Caller] = c
		}
		keys := []string{}
		for k := range unique {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			c := unique[key]
			probes := map[string]assertionFact{}
			for _, a := range assertionFacts(f) {
				if a.Callee != c.Caller {
					continue
				}
				for _, arg := range a.Arguments {
					e, err := parser.ParseExpr(arg)
					if err != nil {
						t.Fatal("unsupported probe argument")
					}
					if _, ok := e.(*ast.BasicLit); !ok {
						t.Fatal("probe supports literal arguments only")
					}
				}
				probes[a.Function+"("+strings.Join(a.Arguments, ",")+")"] = a
			}
			probeKeys := []string{}
			for k := range probes {
				probeKeys = append(probeKeys, k)
			}
			sort.Strings(probeKeys)
			for _, pk := range probeKeys {
				a := probes[pk]
				for _, bits := range [][2]bool{{false, false}, {true, false}, {false, true}, {true, true}} {
					root := t.TempDir()
					write := func(name, content string) {
						p := filepath.Join(root, name)
						if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(p, []byte(content), 0600); err != nil {
							t.Fatal(err)
						}
					}
					write("go.mod", "module fixture\n\ngo 1.23.0\n")
					for _, file := range f.Files {
						if strings.HasSuffix(*file.NewPath, "_test.go") {
							continue
						}
						after := file.ID == c.Callee && bits[0] || file.ID == c.Caller && bits[1]
						write(*file.NewPath, versionSource(file.RawDiff, after))
					}
					pkg := path.Dir(*f.Files[byID[c.Caller]].NewPath)
					write("probe/main.go", fmt.Sprintf("package main\nimport(\"encoding/json\";\"os\";target %q)\nfunc main(){if err:=json.NewEncoder(os.Stdout).Encode(target.%s(%s));err!=nil{panic(err)}}\n", "fixture/"+pkg, a.Function, strings.Join(a.Arguments, ",")))
					callCtx, stop := context.WithTimeout(ctx, 30*time.Second)
					cmd := exec.CommandContext(callCtx, "go", "run", "./probe")
					cmd.Dir = root
					cmd.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local")
					start := time.Now()
					out, err := cmd.Output()
					wall := time.Since(start).Seconds()
					stop()
					if err != nil {
						t.Fatalf("synthetic runtime probe failed: %v", err)
					}
					if !json.Valid(out) {
						t.Fatal("invalid probe result")
					}
					all = append(all, interactionObservation{f.Name, c.Callee, c.Caller, a.Function, a.Arguments, bits[0], bits[1], json.RawMessage(strings.TrimSpace(string(out))), wall})
				}
			}
		}
	}
	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if dest := os.Getenv("BENCHMARK149_PROBE_ARTIFACT"); dest != "" {
		if err := os.WriteFile(dest, append(data, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("synthetic runtime probe: %d observations, model_calls=0", len(all))
}
