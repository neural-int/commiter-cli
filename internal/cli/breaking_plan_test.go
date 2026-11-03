package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/historyattribution"
	"github.com/natsuki0413/commiter-cli/internal/mlxmodel"
)

// Exercise generateCommitPlan without manually supplying planner Evidence.
// Only model assets, helper responses and optional history are simulated.
func TestNormalCLICompletesGroundedBreakingPlan(t *testing.T) {
	for _, ref := range []string{"git:F1:B1", "none", "unresolved", "git:F9:B1"} {
		t.Run(ref, func(t *testing.T) {
			root := cliRepository(t)
			before := "package sample\nimport \"flag\"\nfunc Flags() { flag.Bool(\"legacy\",false,\"legacy\"); flag.Bool(\"current\",false,\"current\") }\n"
			after := "package sample\nimport \"flag\"\nfunc Flags() { flag.Bool(\"current\",false,\"current\") }\n"
			cliWrite(t, root, "flags.go", before, 0600)
			cliGit(t, root, "add", "flags.go")
			cliGit(t, root, "commit", "-m", "base")
			cliWrite(t, root, "flags.go", after, 0600)
			snapshot, err := gitstate.Collect(root, gitstate.Options{})
			if err != nil {
				t.Fatal(err)
			}
			values := config.Defaults().Values
			store := mlxmodel.Store{Root: t.TempDir()}
			spec := mlxmodel.Spec{Repo: values.Model, Revision: values.ModelRevision, Quantization: values.ModelQuantization}
			model, err := store.Destination(spec)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.MkdirAll(model, 0700); err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(model, "config.json"), []byte("{}"), 0600); err != nil {
				t.Fatal(err)
			}
			metadata, _ := json.Marshal(mlxmodel.Plan{Spec: spec, Files: []mlxmodel.File{{Path: "config.json", Size: 2}}})
			if err = os.WriteFile(filepath.Join(model, "commiter-model.json"), metadata, 0600); err != nil {
				t.Fatal(err)
			}
			oldStore, oldHelper, oldHistory := newMLXModelStore, mlxHelperPath, collectAttributedHistory
			t.Cleanup(func() { newMLXModelStore = oldStore; mlxHelperPath = oldHelper; collectAttributedHistory = oldHistory })
			newMLXModelStore = func() (mlxmodel.Store, error) { return store, nil }
			collectAttributedHistory = func(context.Context, historyattribution.Request) (historyattribution.Result, error) {
				return historyattribution.Result{Status: "unavailable"}, nil
			}
			helper := filepath.Join(t.TempDir(), "helper")
			script := `#!/bin/sh
set -eu
while IFS= read -r request; do
case "$request" in
 *'"generation_profile":"bounded-grouping"'*) profile=bounded-grouping; result='{"F1":"G001"}' ;;
 *'"generation_profile":"bounded-category"'*)
 case "$request" in *'git:F1:B1'*) ;; *) exit 2 ;; esac
 profile=bounded-category; result='{"G001":{"type":"feat","breaking_evidence_ref":"REFERENCE"}}' ;;
 *'"generation_profile":"bounded-text"'*)
 case "$request" in *'breaking_evidence_candidates'*|*'Observed current Git change'*) exit 3 ;; esac
 profile=bounded-text; result='{"G001":{"scope":"cli","summary":"remove legacy option"}}' ;;
 *) exit 1 ;;
esac
escaped=$(printf '%s' "$result" | sed 's/"/\\"/g')
printf '{"ok":true,"stop_reason":"completed","generation_profile":"%s","generated_json":"%s"}\n' "$profile" "$escaped"
done
`
			script = strings.ReplaceAll(script, "REFERENCE", ref)
			if err = os.WriteFile(helper, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			mlxHelperPath = func() (string, error) { return helper, nil }
			plan, err := generateCommitPlan(context.Background(), root, snapshot, values, "")
			if ref == "unresolved" || ref == "git:F9:B1" {
				if err == nil || len(plan.Commits) > 0 {
					t.Fatal("unsupported evidence accepted", plan, err)
				}
			} else if err != nil || len(plan.Commits) != 1 || plan.Commits[0].Breaking != (ref != "none") {
				t.Fatal("normal CLI breaking plan failed", plan, err)
			}
			if err = verifyAttributionSnapshot(context.Background(), root, snapshot, values); err != nil {
				t.Fatal(err)
			}
		})
	}
}
