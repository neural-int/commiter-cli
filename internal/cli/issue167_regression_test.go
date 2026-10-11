package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/mlxmodel"
	"github.com/natsuki0413/commiter-cli/internal/planning"
)

// Explicit local research entry point. Default repository tests never infer.
// It invokes the actual product adapter, without executing the doctor's extra
// three probes or mutating the selected changes into commits.
func TestIssue167RegressionProductPath(t *testing.T) {
	name, helper, cache, output := os.Getenv("COMMITER_167_REGRESSION_FIXTURE"), os.Getenv("COMMITER_CANDIDATE_SMOKE_HELPER"), os.Getenv("COMMITER_CANDIDATE_SMOKE_CACHE"), os.Getenv("COMMITER_167_REGRESSION_OUT")
	if name == "" {
		t.Skip("explicit local regression invocation only")
	}
	if helper == "" || cache == "" || output == "" {
		t.Fatal("explicit helper/cache/output required")
	}
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_OPTIONAL_LOCKS", "0")
	path, before, after := "sample.go", "package sample\nfunc value() int { return 1 }\n", "package sample\nfunc value() int { return 2 }\n"
	if name == "one-independent" {
		path = "settings/limit0.go"
		before = "package settings\n\nfunc WithinLimit0(value int) bool { return value < 10 }\n"
		after = strings.Replace(before, "value < ", "value <= ", 1)
	} else if name != "smoke-143" {
		t.Fatal("unknown authored fixture")
	}
	repo := cliRepository(t)
	cliWrite(t, repo, path, before, 0644)
	cliGit(t, repo, "add", path)
	cliGit(t, repo, "commit", "-m", "base")
	cliWrite(t, repo, path, after, 0644)
	snapshot, err := gitstate.Collect(repo, gitstate.Options{})
	if err != nil {
		t.Fatal("fixture collection failed")
	}
	// Build a receipt using the same functions as generateCommitPlan. Its
	// relation context can differ from benchmark Prepared while actual requests
	// are compared separately at the helper boundary.
	results, _, _, relations, err := analyzeForPlanningWithStatsAndRelationsContext(context.Background(), repo, snapshot)
	if err != nil {
		t.Fatal("fixture analysis failed")
	}
	doc, err := contextinput.Build(snapshot, results)
	if err != nil {
		t.Fatal("fixture input failed")
	}
	attachRelationContext(&doc, snapshot.Changes, relations)
	oldStore, oldHelper := newMLXModelStore, mlxHelperPath
	t.Cleanup(func() { newMLXModelStore, mlxHelperPath = oldStore, oldHelper })
	newMLXModelStore = func() (mlxmodel.Store, error) { return mlxmodel.Store{Root: cache}, nil }
	mlxHelperPath = func() (string, error) { return helper, nil }
	statusBefore := cliGitOutput(t, repo, "status", "--porcelain")
	repoBefore := issue167RepositoryHash(t, repo)
	indexBefore, err := os.ReadFile(filepath.Join(repo, ".git", "index"))
	if err != nil {
		t.Fatal("fixture index unavailable")
	}
	started := time.Now()
	plan, planErr := generateCommitPlan(context.Background(), repo, snapshot, config.Defaults().Values, "")
	wall := time.Since(started).Seconds()
	indexAfter, err := os.ReadFile(filepath.Join(repo, ".git", "index"))
	if err != nil {
		t.Fatal("fixture index unavailable")
	}
	wt, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal("fixture content unavailable")
	}
	unchanged := statusBefore == cliGitOutput(t, repo, "status", "--porcelain") && string(indexBefore) == string(indexAfter) && string(wt) == after && repoBefore == issue167RepositoryHash(t, repo)
	status := "completed"
	codes := []string{}
	if planErr != nil {
		status = "planner_failed"
		tokens := strings.FieldsFunc(planErr.Error(), func(r rune) bool { return !(r >= 'a' && r <= 'z') && r != '_' })
		for _, code := range []string{"invalid_json", "invalid_schema", "invalid_type", "invalid_scope", "invalid_summary_language", "invalid_summary", "invalid_assignment", "incomplete_output", "sensitive_output", "unresolved_breaking_evidence"} {
			for _, token := range tokens {
				if token == code {
					codes = append(codes, code)
					break
				}
			}
		}
		if len(codes) == 0 {
			codes = append(codes, "unclassified_planner_failure")
		}
	}
	valid := false
	if planErr == nil {
		data, _ := json.Marshal(plan)
		_, v := planning.Validate(data, []string{"F001"}, planning.SensitiveValues{}, planning.English)
		valid = len(v) == 0 && len(plan.Commits) == 1 && strings.Join(plan.Commits[0].FileIDs, ",") == "F001"
	}
	hash := func(data []byte) string { d := sha256.Sum256(data); return hex.EncodeToString(d[:]) }
	docBytes, _ := json.Marshal(doc)
	data, _ := json.MarshalIndent(map[string]any{"fixture": name, "route": "product-cli-three-phase", "status": status, "failure_codes": codes, "wall_seconds": wall, "final_validator_pass": valid, "git_unchanged": unchanged, "groups": len(plan.Commits), "prepared_document_sha256": hash(docBytes), "fixture_before_sha256": hash([]byte(before)), "fixture_after_sha256": hash([]byte(after)), "model_inference": os.Getenv("COMMITER_167_PROXY_MODE") == "real"}, "", "  ")
	if os.WriteFile(output, append(data, '\n'), 0600) != nil {
		t.Fatal("cannot write numeric receipt")
	}
	if !unchanged {
		t.Fatal("regression modified selected Git state")
	}
}

func issue167RepositoryHash(t *testing.T, root string) string {
	t.Helper()
	items := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unexpected synthetic symlink")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		h := sha256.Sum256(data)
		items[rel] = fmt.Sprintf("%o:%x", info.Mode(), h)
		return nil
	})
	if err != nil {
		t.Fatal("fixture digest failed")
	}
	data, _ := json.Marshal(items)
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
